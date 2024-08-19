var="GOARCH='amd64' GOOS='windows' CGO_ENABLED=1 CC='/usr/bin/x86_64-w64-mingw32-gcc'"
build_file="ecosys_windows_x64.exe"
eval "$var go build -buildmode=exe -o $build_file main.go"

var=""
build_file="ecosys_linux_x64"
eval "$var go build -o $build_file main.go"
