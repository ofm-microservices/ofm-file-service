# OFM File Service

## Purpose

The File Service owns file metadata, ownership, lifecycle, and object-storage operations. It is the source of truth for file records and stores binary data in RustFS. Status: active.

## Boundaries and flow

Clients and other services call the gRPC file API for upload, lookup, URL, and delete operations. The service stores the object and authoritative metadata, then compensates by removing the object if metadata persistence fails. It does not own gig media, order attachments, chat messages, or user profiles.

PostgreSQL is the authoritative metadata store. Committed outbox changes are captured by Debezium and published through Kafka for projections. Processing is idempotent and acknowledgments occur after the target transaction succeeds.

## Configuration

.env.example groups are DB_*, MIGRATIONS_*, RUSTFS_*, gRPC/metrics listeners, broker settings, and observability. Database values select file storage; RustFS values select endpoint, bucket, and credentials; listener values select the gRPC port. Never commit .env.

## Local development

    cp .env.example .env
    just run
    go test ./...

## Build and operations

Dockerfile builds ofm/file-service:<tag>. ofm-infra deploys the service and provides RustFS, PostgreSQL, brokers, and observability. Diagnose object and metadata compensation, outbox/CDC, Kafka lag, and projection audit state.

