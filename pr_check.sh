#!/bin/bash

# --------------------------------------------
# Options that must be configured by app owner
# --------------------------------------------
#export APP_NAME="entitlements"  # name of app-sre "application" folder this component lives in
#export COMPONENT_NAME="entitlements-api-go"  # name of app-sre "resourceTemplate" in deploy.yaml for this component
#export IMAGE="quay.io/cloudservices/entitlements-api-go"  # the image location on quay

export GOROOT="/opt/go/1.20.10"
export PATH="${GOROOT}/bin:${PATH}"

echo "*** GO version ***"
go version

# Install bonfire repo/initialize
CICD_URL=https://raw.githubusercontent.com/RedHatInsights/bonfire/master/cicd
curl -s $CICD_URL/bootstrap.sh > .cicd_bootstrap.sh && source .cicd_bootstrap.sh

# Build the image and push to quay
source $CICD_ROOT/build.sh

make test-all
if [ $? != 0 ]; then
    exit 1
fi

cp coverage.txt $ARTIFACTS_DIR

# manually paste dummy PR Check results until we setup iqe tests to run in build and they post results
# see here for an example: https://github.com/RedHatInsights/insights-ingress-go/blob/master/pr_check.sh#L23-L25
# with cji_smoke_test and post_test_results, we can run iqe tests and they will post results to this dir
cat << EOF > $ARTIFACTS_DIR/junit-dummy.xml
<testsuite tests="1">
    <testcase classname="dummy" name="dummytest"/>
</testsuite>
EOF
