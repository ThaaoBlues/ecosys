
# to cross-compile on linux for windows with the webview module, we must before do some tricks :
# install mingw
# sudo cp /usr/share/mingw-w64/include/eventtoken.h /home/thaao/go/pkg/mod/github.com/webview/webview_go@v0.0.0-20240831120633-6173450d4dd6/libs/mswebview2/include/EventToken.h
var="GOARCH='amd64' GOOS='windows' CGO_ENABLED=1 CC='/usr/bin/x86_64-w64-mingw32-gcc' CXX='/usr/bin/x86_64-w64-mingw32-g++'"
build_file="ecosys_windows_x64.exe"
eval "$var go build -buildmode=exe -o $build_file main.go"

var=""
build_file="ecosys_linux_x64"
eval "$var go build -o $build_file main.go"
