# usage: up <file>
up() { SERVER="http://<server-IP>:8080/upload"; command -v curl >/dev/null || { sudo apt update && sudo apt install -y curl; } && curl -F "file=@$1" "$SERVER" || echo "Upload failed"; }
