SCRIPT_DIR = './backend/installer_kit/installer_scripts'

.PHONY: run-installer
run-installer:
	$(SCRIPT_DIR)/installer.sh

.PHONY: destroy-infra
destroy-infra:
	$(SCRIPT_DIR)/destroy-infra.sh

.PHONY: setup-agent
setup-agent:
	./agent_pack/setup_agent_app.sh