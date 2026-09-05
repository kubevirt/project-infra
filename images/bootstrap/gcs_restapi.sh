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

# Function to validate GOOGLE_APPLICATION_CREDENTIALS and set access_token global variable
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

get_auth_header() {
    local auth=${1:-"true"}

    if [ "$auth" == "false" ]; then
        echo ""
        return 0
    fi

    get_access_token || exit 1
    echo "Authorization: Bearer $access_token"
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

    auth_header=$(get_auth_header) || exit 1

    upload_response=$(curl -X POST \
      --data-binary @"$source_file" \
      -H "$auth_header" \
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
    local auth="$3"

    auth_header=$(get_auth_header "$auth") || exit 1

    local stat_response
    stat_response=$(curl --silent --show-error --fail-with-body -X GET \
      ${auth_header:+-H "$auth_header"} \
      "${BASE_URL}/storage/v1/b/$bucket_name/o/$gcs_file_path")
    local curl_exit=$?

    if [ "$curl_exit" -ne 0 ]; then
        return 1
    fi

    if ! echo "$stat_response" | jq -e '.name' > /dev/null 2>&1; then
        echo "Warning: stat succeeded but response has no .name for $2: $stat_response" >&2
        return 1
    fi

    return 0
}

# Function to read the content of a file from GCS
cat_gcs_file() {
    local bucket_name="$1"
    local gcs_file_path=$(urlencode_path "$2")
    local auth="$3"

    auth_header=$(get_auth_header "$auth") || exit 1

    local tmpfile
    tmpfile=$(mktemp) || { echo "Error: mktemp failed" >&2; return 1; }

    local http_code
    http_code=$(curl --silent --show-error --fail-with-body --output "$tmpfile" --write-out '%{http_code}' -X GET \
      ${auth_header:+-H "$auth_header"} \
      -H "Cache-Control: no-cache" \
      "${BASE_URL}/storage/v1/b/$bucket_name/o/$gcs_file_path?alt=media&ignoreCache=1")
    local curl_exit=$?

    if [ "$curl_exit" -ne 0 ]; then
        echo "Error: HTTP ${http_code:-?} for $2 (curl exit $curl_exit)" >&2
        if [ -s "$tmpfile" ]; then
            echo "Response body: $(cat "$tmpfile")" >&2
        fi
        rm -f "$tmpfile"
        return 1
    fi

    if [ ! -s "$tmpfile" ]; then
        echo "Error: HTTP 200 but empty body for $2" >&2
        rm -f "$tmpfile"
        return 1
    fi

    local cat_exit=0
    cat "$tmpfile" || cat_exit=$?
    rm -f "$tmpfile"
    return "$cat_exit"
}

# Function to delete a file from GCS
rm_gcs_file() {
    local bucket_name="$1"
    local gcs_file_path=$(urlencode_path "$2")

    auth_header=$(get_auth_header) || exit 1

    delete_response=$(curl -s -X DELETE \
      -H "$auth_header" \
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