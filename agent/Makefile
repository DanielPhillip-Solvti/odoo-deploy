build-agent:
	rm -rf build/*
	mkdir -p build
	go build -o ./build/agent .
	./build/agent --extract-assets ./build

HOST_IP := $(shell ip -4 route get 1 | grep -oP 'src \K\S+' 2>/dev/null || hostname -I | awk '{print $$1}' 2>/dev/null || echo "127.0.0.1")

write-debug-env:
	@echo "ODOO_URL=http://localhost:8169" > ./build/.env
	@echo "API_KEY=generated_api_key1" >> ./build/.env
	@echo "REPO_URL=https://github.com/DanielPhillip-Solvti/odoo-deploy" >> ./build/.env
	@echo "HEARTBEAT_FILE=${PWD}/build/heartbeat.json" >> ./build/.env
	@echo "WS_PORT=9876" >> ./build/.env
	@echo "PUBLIC_URL=${HOST_IP}" >> ./build/.env
	@echo "POSTGRES_USER=odoo" >> ./build/.env
	@echo "POSTGRES_PASSWORD=odoo" >> ./build/.env
	@echo "AGENT_WS_ADDR=${HOST_IP}" >> ./build/.env

debug-build: build-agent write-debug-env