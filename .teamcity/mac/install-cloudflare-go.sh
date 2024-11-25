rm -rf /tmp/go
export GOCACHE=/tmp/gocache
rm -rf $GOCACHE

./.teamcity/install-khulnasoft-go.sh

export PATH="/tmp/go/bin:$PATH"
go version
which go
go env