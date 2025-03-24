# up <file>
up() { command -v curl >/dev/null || sudo apt update && sudo apt install -y curl; curl -F "file=@$1" http://<server-IP>:8080/upload; }
