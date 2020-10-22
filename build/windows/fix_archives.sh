#!/bin/bash
set -e
#
#
# Gets dist/zip_dirty created by Goreleaser and reorganize inside files
#
#
PROJECT_PATH=$1

for zip_dirty in $(find dist -regex ".*_dirty\.\(zip\)");do
  zip_file_name=${zip_dirty:5:${#zip_dirty}-(5+10)} # Strips begining and end chars
  ZIP_CLEAN="${zip_file_name}.zip"
  ZIP_TMP="dist/zip_temp"
  ZIP_CONTENT_PATH="${ZIP_TMP}/${zip_file_name}_content"

  mkdir -p "${ZIP_CONTENT_PATH}"

  AGENT_DIR_IN_ZIP_PATH="${ZIP_CONTENT_PATH}/New Relic/newrelic-infra/newrelic-integrations/"
  CONF_IN_ZIP_PATH="${ZIP_CONTENT_PATH}/New Relic/newrelic-infra/integrations.d/"

  mkdir -p "${AGENT_DIR_IN_ZIP_PATH}/bin"
  mkdir -p "${CONF_IN_ZIP_PATH}"

  echo "===> Decompress ${zip_file_name} in ${ZIP_CONTENT_PATH}"
  unzip ${zip_dirty} -d ${ZIP_CONTENT_PATH}

  echo "===> Move files inside ${zip_file_name}"
  mv ${ZIP_CONTENT_PATH}/nri-${INTEGRATION}.exe "${AGENT_DIR_IN_ZIP_PATH}/bin"
  #mv ${ZIP_CONTENT_PATH}/${INTEGRATION}-win-definition.yml "${AGENT_DIR_IN_ZIP_PATH}"
  #mv ${ZIP_CONTENT_PATH}/${INTEGRATION}-win-config.yml.sample "${CONF_IN_ZIP_PATH}"

  echo "===> Creating zip ${ZIP_CLEAN}"
  cd "${ZIP_CONTENT_PATH}"
  zip -r ../${ZIP_CLEAN} .
  cd $PROJECT_PATH
  echo "===> Moving zip ${ZIP_CLEAN}"
  mv "${ZIP_TMP}/${ZIP_CLEAN}" dist/
  echo "===> Cleaning dirty zip ${zip_dirty}"
  rm "${zip_dirty}"
done