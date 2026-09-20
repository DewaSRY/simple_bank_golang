#!/bin/bash
set -euxo pipefail

dnf update -y
dnf install -y docker
systemctl enable --now docker
usermod -aG docker ec2-user

# docker compose plugin — Amazon Linux 2023's `docker` package doesn't bundle
# it. Defaults to a pinned release (var.docker_compose_version) for
# reproducibility; set that variable to "latest" to opt back into resolving
# GitHub's current release at boot instead. Either way this now uses
# --retry and an explicit `docker compose version` check afterward — the
# previous always-latest lookup had neither, so a transient network blip or
# GitHub rate limit during boot silently left docker-compose missing/empty
# and the whole stack never started, with `set -e` aborting the script right
# there before `docker ps` ever had anything to show.
COMPOSE_VERSION="${docker_compose_version}"
if [ "$COMPOSE_VERSION" = "latest" ]; then
  COMPOSE_VERSION=$(curl -fsSL --retry 5 --retry-delay 5 --retry-connrefused \
    https://api.github.com/repos/docker/compose/releases/latest \
    | grep -m1 '"tag_name"' | cut -d '"' -f4)
  if [ -z "$COMPOSE_VERSION" ]; then
    echo "ERROR: failed to resolve the latest docker compose release from GitHub's API" >&2
    exit 1
  fi
fi

mkdir -p /usr/local/lib/docker/cli-plugins
curl -fsSL --retry 5 --retry-delay 5 --retry-connrefused \
  "https://github.com/docker/compose/releases/download/$COMPOSE_VERSION/docker-compose-linux-x86_64" \
  -o /usr/local/lib/docker/cli-plugins/docker-compose
chmod +x /usr/local/lib/docker/cli-plugins/docker-compose

# Fail loudly here, with a clear error in cloud-init-output.log, rather than
# silently reaching `docker compose up -d` with a broken/missing plugin.
docker compose version

mkdir -p /opt/core-service

cat > /opt/core-service/app.env <<EOF
DB_DRIVER=${db_driver}
DB_SOURCE=${db_source}
SERVER_ADDRESS=0.0.0.0:${app_port}
JWT_SECRET_KEY=${jwt_secret_key}
JWT_ACCESS_TOKEN_DURATION=${jwt_access_token_duration}
CORS_ALLOWED_ORIGINS=${cors_allowed_origins}
RATE_LIMIT_ENABLED=${rate_limit_enabled}
RATE_LIMIT_REQUESTS_PER_SECOND=${rate_limit_rps}
RATE_LIMIT_BURST=${rate_limit_burst}
EOF
chmod 600 /opt/core-service/app.env

echo '${nginx_conf_base64}' | base64 -d > /opt/core-service/nginx.conf
chmod 644 /opt/core-service/nginx.conf

echo '${docker_compose_yml_base64}' | base64 -d > /opt/core-service/docker-compose.yml
chmod 644 /opt/core-service/docker-compose.yml

# Idempotent: pull + up -d each time, safe to rerun over SSH on an
# already-running instance (see docs/TERRAFORM_EC2_DEPLOY.md Section 7).
cd /opt/core-service
docker compose pull
docker compose up -d
