# Rendered by templatefile() in main.tf from variables (nginx_port,
# app_port, rate_limit_rps, rate_limit_burst), then shipped to the instance
# base64-encoded inside user_data.sh.tpl. Mounted at
# /etc/nginx/conf.d/default.conf, replacing the stock nginx image config.
#
# core-service itself is not published on the instance's public interface —
# nginx is the only public entrypoint, reaching core-service by container
# name over the shared docker network (see user_data.sh.tpl).

limit_req_zone $binary_remote_addr zone=core_service:10m rate=${rate_limit_rps}r/s;

server {
    listen ${nginx_port};
    server_name _;

    location / {
        limit_req zone=core_service burst=${rate_limit_burst} nodelay;
        limit_req_status 429;

        proxy_pass http://core-service:${app_port};
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
