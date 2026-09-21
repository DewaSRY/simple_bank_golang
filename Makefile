CORE_SERVICE_DIR := apps/core-service

TF_DIR := infra/terraform
TF_KEY := $(TF_DIR)/core-service-key.pem

tf-init:
	terraform -chdir=$(TF_DIR) init

tf-fmt:
	terraform -chdir=$(TF_DIR) fmt

tf-validate:
	terraform -chdir=$(TF_DIR) validate

tf-plan:
	terraform -chdir=$(TF_DIR) plan

tf-apply:
	terraform -chdir=$(TF_DIR) apply

tf-output:
	terraform -chdir=$(TF_DIR) output

tf-destroy:
	terraform -chdir=$(TF_DIR) destroy

# Re-runs the exact first-boot script (docker compose stack: core-service +
# nginx, see infra/terraform/docker-compose.prod.yaml) against the
# already-running instance over SSH. EC2 only executes user_data
# automatically on an instance's first boot, so this is how an existing
# instance picks up a new image, a new nginx rate limit, a new CORS origin,
# etc. Idempotent (docker compose pull + up -d each time) — safe to re-run.
# See infra/terraform/outputs.tf's rendered_user_data and
# apps/core-service/docs/TERRAFORM_EC2_DEPLOY.md Section 7.
tf-redeploy:
	$(eval EC2_IP := $(shell terraform -chdir=$(TF_DIR) output -raw public_ip))
	terraform -chdir=$(TF_DIR) output -raw rendered_user_data | ssh -i $(TF_KEY) ec2-user@$(EC2_IP) 'sudo bash -s'

# Builds + pushes the core-service prod image, then redeploys the EC2
# instance against it (and against whatever nginx/rate-limit settings are
# currently in infra/terraform).
deploy:
	$(MAKE) -C $(CORE_SERVICE_DIR) docker-dep-build docker-dep-push
	$(MAKE) tf-redeploy

.PHONY: tf-init tf-fmt tf-validate tf-plan tf-apply tf-output tf-destroy tf-redeploy deploy
