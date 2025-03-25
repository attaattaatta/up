# usage: up <file>
up() { SERVER="http://<server-IP>:5555/upload"; command -v curl >/dev/null || { sudo apt update && sudo apt install -y curl; } && curl -F "file=@$1" "$SERVER" || echo "Upload failed"; }
