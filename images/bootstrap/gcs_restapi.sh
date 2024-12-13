#!/usr/bin/env bash
# Copyright 2024 The KubeVirt Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

BASE_URL="https://storage.googleapis.com"

# Function to validate GOOGLE_APPLICATION_CREDENTIALS and and set access_token global variable
get_access_token() {
    if [ -z "${GOOGLE_APPLICATION_CREDENTIALS}" ]; then
        echo "GOOGLE_APPLICATION_CREDENTIALS is not set. Please set it to the path of your service account key file."
        exit 1
    fi

    # Reuse the token if it's still valid
    if [ -n "$access_token" ] && [ "$(date +%s)" -lt "$token_expiry" ]; then
        return 0
    fi

    local sa_email=$(jq -r '.client_email' "$GOOGLE_APPLICATION_CREDENTIALS")
    local sa_key=$(jq -r '.private_key' "$GOOGLE_APPLICATION_CREDENTIALS")
    local jwt_header=$(echo -n '{"alg":"RS256","typ":"JWT"}' | base64 -w 0 | tr '+/' '-_' | tr -d '=')
    local jwt_claim=$(echo -n '{"iss":"'$sa_email'","scope":"https://www.googleapis.com/auth/cloud-platform","aud":"https://oauth2.googleapis.com/token","exp":'$(($(date +%s) + 3600))',"iat":'$(date +%s)'}' | base64 -w 0 | tr '+/' '-_' | tr -d '=')
    local jwt_signature=$(echo -n "$jwt_header.$jwt_claim" | openssl dgst -binary -sha256 -sign <(echo "$sa_key") | base64 -w 0 | tr '+/' '-_' | tr -d '=')
    local jwt="$jwt_header.$jwt_claim.$jwt_signature"

    local response=$(curl -s -X POST https://oauth2.googleapis.com/token \
         -H "Content-Type: application/x-www-form-urlencoded" \
         -d "grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer&assertion=$jwt")

    access_token=$(echo "$response" | jq -r '.access_token')
    token_expiry=$(($(date +%s) + 3600)) # 1 hour expiry

    if [ -z "$access_token" ]; then
        echo "Failed to obtain access token. Check your service account key file."
        exit 1
    fi
}

urlencode_path() {
    local path="$1"
    echo "$path" | sed 's/\//%2F/g'
}

# Function to upload a file to Google Cloud Storage
upload_to_gcs() {
    local source_file="$1"
    local bucket_name="$2"
    local destination_blob=$(urlencode_path "$3")
    local content_type="application/octet-stream"

    get_access_token

    upload_response=$(curl -X POST \
      --data-binary @"$source_file" \
      -H "Authorization: Bearer $access_token" \
      -H "Content-Type: $content_type" \
      "${BASE_URL}/upload/storage/v1/b/$bucket_name/o?uploadType=media&name=$destination_blob")

    if echo "$upload_response" | jq -e '.name' > /dev/null; then
       echo "File $source_file uploaded successfully as $destination_blob"
       return 0
    else
       echo "Upload failed. Response:"
       echo "$upload_response" | jq '.'
       return 1
    fi
}

# Function to check if a file exists in GCS
stat_gcs_file() {
    local bucket_name="$1"
    local gcs_file_path=$(urlencode_path "$2")

    get_access_token

    stat_response=$(curl -s -X GET \
      -H "Authorization: Bearer $access_token" \
      "${BASE_URL}/storage/v1/b/$bucket_name/o/$gcs_file_path")

    if echo "$stat_response" | jq -e '.error' > /dev/null; then
        return 1
    else
        return 0
    fi
}

# Function to read the content of a file from GCS
cat_gcs_file() {
    local bucket_name="$1"
    local gcs_file_path=$(urlencode_path "$2")

    get_access_token

    file_content=$(curl -s -X GET \
      -H "Authorization: Bearer $access_token" \
      "${BASE_URL}/storage/v1/b/$bucket_name/o/$gcs_file_path?alt=media")

    if [ -z "$file_content" ]; then
        echo "Error: No content received"
        return 1
    else
        echo "$file_content"
        return 0
    fi
}

# Function to delete a file from GCS
rm_gcs_file() {
    local bucket_name="$1"
    local gcs_file_path=$(urlencode_path "$2")

    get_access_token

    delete_response=$(curl -s -X DELETE \
      -H "Authorization: Bearer $access_token" \
      "${BASE_URL}/storage/v1/b/$bucket_name/o/$gcs_file_path")

    if [ -z "$delete_response" ]; then
        echo "File $gcs_file_path deleted successfully."
        return 0
    else
        echo "Failed to delete file. Response:"
        echo "$delete_response" | jq '.'
        return 1
    fi
}