#!/usr/bin/env bash
# サーバー上で実行し、送られてきた server.new に置き換えて API を再起動する。
# GitHub Actions から `ssh <サーバー> 'bash -s' < api/deploy/remote-deploy.sh` で実行する。
# /health が通らなければ、元のバイナリに戻して失敗にする。
set -euo pipefail

APP_DIR="$HOME/services/meshi-diary"
SERVICE=meshi-diary-api
HEALTH_URL=http://127.0.0.1:8080/health

# 再起動とログの表示は sudo が要る。CI ではパスワードを入れられないので、-n でパスワードを聞かずに失敗させる。
# パスワードなしで実行できるコマンドは deploy/sudoers-meshi-diary-api で決めている（引数も完全に一致させる）
restart() {
	sudo -n systemctl restart "$SERVICE"
}

cd "$APP_DIR"

wait_healthy() {
	for _ in $(seq 1 15); do
		if curl -fsS "$HEALTH_URL" >/dev/null; then
			return 0
		fi
		sleep 1
	done
	return 1
}

chmod 755 server.new
# 動いているバイナリに直接上書きすると Text file busy で失敗するので、mv で置き換える
if [ -f server ]; then
	cp -p server server.prev
fi
mv server.new server
restart

if wait_healthy; then
	echo "deployed: $(sha256sum server | cut -d' ' -f1)"
	exit 0
fi

echo "health check failed" >&2
sudo -n journalctl -u "$SERVICE" -n 50 --no-pager >&2 || true

if [ -f server.prev ]; then
	echo "rolling back to the previous binary" >&2
	mv server.prev server
	restart
	wait_healthy || echo "previous binary is not healthy either" >&2
fi
exit 1
