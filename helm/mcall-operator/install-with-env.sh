#!/bin/bash

# Set passwords using environment variables
export POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-}"
export MYSQL_PASSWORD="${MYSQL_PASSWORD:-}"
export ELASTICSEARCH_PASSWORD="${ELASTICSEARCH_PASSWORD:-}"

# Install with Helm
helm install mcall-crd mcall/mcall-crd \
  --namespace mcall-system \
  --create-namespace \
  --values values.yaml \
  --set logging.postgresql.password="$POSTGRES_PASSWORD" \
  --set logging.mysql.password="$MYSQL_PASSWORD" \
  --set logging.elasticsearch.password="$ELASTICSEARCH_PASSWORD"

echo "✅ mcall-operator installed successfully!"
echo "📝 To check logs: kubectl logs -n mcall-system -l app.kubernetes.io/name=tz-mcall-operator"
