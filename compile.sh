if test "win" = $1; then
    var="GOARCH='amd64' GOOS='windows' CGO_ENABLED=1 CC='/usr/bin/x86_64-w64-mingw32-gcc'"

else
    var=""
fi

eval "$var go build main.go"
