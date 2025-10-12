.PHONY: help install build clean docker-build docker-push deploy test

# Variables
IMAGE_NAME := doohee323/mcall-operator-mcp-server
IMAGE_TAG := latest
NAMESPACE := mcall-system

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

install: ## Install dependencies
	npm install

build: ## Build TypeScript code
	npm run build

clean: ## Clean build artifacts
	rm -rf dist node_modules

dev: ## Run in development mode
	npm run dev

docker-build: ## Build Docker image
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

docker-push: ## Push Docker image to registry
	docker push $(IMAGE_NAME):$(IMAGE_TAG)

docker-build-push: docker-build docker-push ## Build and push Docker image

deploy: ## Deploy to Kubernetes using kubectl
	kubectl apply -f k8s/

undeploy: ## Remove from Kubernetes
	kubectl delete -f k8s/

helm-install: ## Install using Helm with MCP server enabled
	cd ../helm/mcall-operator && \
	helm upgrade --install mcall-operator . \
		--namespace $(NAMESPACE) \
		--create-namespace \
		--set mcpServer.enabled=true \
		--wait

helm-uninstall: ## Uninstall using Helm
	helm uninstall mcall-operator -n $(NAMESPACE)

test: ## Run tests
	npm test

logs: ## Show MCP server logs
	kubectl logs -n $(NAMESPACE) -l app.kubernetes.io/component=mcp-server -f

status: ## Check MCP server status
	kubectl get pods -n $(NAMESPACE) -l app.kubernetes.io/component=mcp-server
	kubectl get svc -n $(NAMESPACE) -l app.kubernetes.io/component=mcp-server
	kubectl get ingress -n $(NAMESPACE) -l app.kubernetes.io/component=mcp-server

