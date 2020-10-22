echo "--- Running tests"

## test everything excluding vendor
go test $(go list ./... | sls -NotMatch '/vendor/')
if (-not $?)
{
    echo "Failed running tests"
    exit -1
}