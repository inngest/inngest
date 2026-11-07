# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements.  See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership.  The ASF licenses this file
# to you under the Apache License, Version 2.0 (the "License");
# you may not use it except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# golangci-lint version (keep in sync with CI and README)
GOLANGCI_LINT_VERSION := v2.12.2

.PHONY: test test-assert test-race lint lint-install integration-setup integration-setup-spark4 integration-test integration-scanner integration-io integration-rest integration-rest-scan-planning integration-spark integration-hadoop integration-down integration-logs docs-gen

test:
	go test -v ./...

test-assert:
	go test -tags=assert -v -run='^(TestInspectFilesTablesEarlyRelease|TestInspectDataFilesEmitsEmptyBatchWhenAllEntriesAreDeleted|TestInspectDataFilesEmptyTableEarlyRelease|TestInspectPositionDeletesEarlyRelease)$$' ./table

test-race:
	go test -race -v ./...

docs-gen:
	go run ./website/gen

lint:
	golangci-lint run --timeout=10m

lint-install:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

integration-setup:
	mkdir -p /tmp/iceberg-hadoop-warehouse
	docker compose -f internal/recipe/docker-compose.yml up -d --build --wait
	docker compose -f internal/recipe/docker-compose.yml exec -T spark-iceberg ipython ./provision.py

integration-setup-spark4:
	mkdir -p /tmp/iceberg-hadoop-warehouse
	SPARK_IMAGE=pyiceberg-spark4 SPARK_DOCKERFILE=Dockerfile.spark4 docker compose -f internal/recipe/docker-compose.yml build spark-iceberg
	SPARK_IMAGE=pyiceberg-spark4 SPARK_DOCKERFILE=Dockerfile.spark4 docker compose -f internal/recipe/docker-compose.yml up -d --wait
	docker compose -f internal/recipe/docker-compose.yml exec -T spark-iceberg ipython ./provision.py

integration-down:
	docker compose -f internal/recipe/docker-compose.yml down -v

integration-logs:
	docker compose -f internal/recipe/docker-compose.yml logs

integration-env:
	@echo "export AWS_S3_ENDPOINT=http://$$(docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' object-store):9000"
	@echo "export AWS_REGION=us-east-1"
	@echo "export SPARK_CONTAINER_ID=$$(docker ps -qf 'name=spark-iceberg')"
	@echo "export DOCKER_API_VERSION=$$(docker version -f '{{.Server.APIVersion}}')"

# Keep the isolated Java scan-planning suite in umbrella CI so its wire
# compatibility coverage runs alongside the other integration tests.
integration-test: integration-scanner integration-io integration-rest integration-rest-scan-planning integration-spark integration-hive integration-hadoop

integration-scanner:
	go test -tags=integration -v -run="^TestScanner" ./table

integration-io:
	go test -tags=integration -v ./io/...

integration-rest:
	go test -tags=integration -v -run="^(TestRestIntegration|TestRestCatalogConformance)$$" ./catalog/rest

integration-rest-scan-planning:
	RUN_INTEGRATION_TESTS=1 go test -tags=integration -v -run="^TestRestIntegration/TestScanPlanningJava" ./catalog/rest

integration-spark:
	go test -tags=integration -v -run="^TestSparkIntegration" ./table

integration-hive:
	go test -tags=integration -v ./catalog/hive/...

integration-hadoop:
	go test -tags=integration -v -run="^TestHadoopIntegration" ./catalog/hadoop/...

integration-hadoop-clean:
	rm -rf /tmp/iceberg-hadoop-warehouse/*
