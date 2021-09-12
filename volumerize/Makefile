default: help

cnf ?= .env
include $(cnf)
export $(shell sed 's/=.*//' $(cnf))



backup:
	docker run --rm \
    --name test-volumerize \
    -v volumerize_backup:/backup \
	-v volumerize_volumerize-cache:/volumerize-cache \
	-v volumerize_test_volume:/source/test_volume:ro \
    -e "TZ=${VOLUMERIZE_TIMEZONE}" \
    -e "VOLUMERIZE_SOURCE=/source" \
    -e "VOLUMERIZE_TARGET=${VOLUMERIZE_TARGET}" \
    -e "VOLUMERIZE_JOBBER_TIME=${VOLUMERIZE_JOBBER_TIME}" \
    -e "VOLUMERIZE_FULL_IF_OLDER_THAN=${VOLUMERIZE_FULL_IF_OLDER_THAN}" \
    -e "AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID}" \
    -e "AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY}" \
    blacklabelops/volumerize backup

restore:
	docker run --rm \
    --name test-volumerize \
    -v volumerize_backup:/backup:ro \
	-v volumerize_test_volume:/source/test_volume \
    -e "TZ=${VOLUMERIZE_TIMEZONE}" \
    -e "VOLUMERIZE_SOURCE=/source" \
    -e "VOLUMERIZE_TARGET=${VOLUMERIZE_TARGET}" \
    -e "VOLUMERIZE_JOBBER_TIME=${VOLUMERIZE_JOBBER_TIME}" \
    -e "VOLUMERIZE_FULL_IF_OLDER_THAN=${VOLUMERIZE_FULL_IF_OLDER_THAN}" \
    -e "AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID}" \
    -e "AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY}" \
    blacklabelops/volumerize restore