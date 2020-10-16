echo "--- Running tests"

$files = go list ./cmd/... ./internal/...
go test $files
if (-not $?)
{
    echo "Failed running tests"
    exit -1
}