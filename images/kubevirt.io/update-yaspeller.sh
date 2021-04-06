#!/bin/bash
# COLORS
RED=`tput setaf 1` 
GREEN=`tput setaf 2`
RESET=`tput sgr0`

YASPELLER_URL="https://raw.githubusercontent.com/kubevirt/project-infra/master/images/yaspeller/.yaspeller.json"
YASPELLER_PATH="/tmp/yaspeller.json"

if [ -f "${YASPELLER_PATH}" ];
then
     rm -f "${YASPELLER_PATH}"
fi
echo "${GREEN}Downloading Dictionary file: ${YASPELLER_URL} ${RESET}"
if [ "`curl ${YASPELLER_URL} -o "${YASPELLER_PATH}" -w '%{http_code}\n' -s`" != "200" ]
then
     echo "${RED}ERROR: Unable to curl yaspeller dictionary file ${RESET}"
     exit 1
fi
cat "${YASPELLER_PATH}" 2>&1 | jq > /dev/null 2>&1 || {
     echo "${RED}ERROR: yaspeller dictionary file does not exist or is invalid json ${RESET}"
     exit 1
}
echo "${GREEN}yaspeller updated ${RESET}"

     





