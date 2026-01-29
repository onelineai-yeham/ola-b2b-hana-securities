# Hana Securities News API - K8S Deployment Guide

## Overview

이 문서는 `hana-news-api` 서비스를 GKE `ola-b2b` 클러스터에 배포하는 방법을 설명합니다.

## Infrastructure

| 항목 | 값 |
|------|-----|
| GCP Project | `finola-global` |
| GKE Cluster | `ola-b2b` (asia-northeast3) |
| Namespace | `hana-securities` |
| Domain | `api.hana-securities.onelineai.com` |
| Static IP | `136.110.137.91` |

## Prerequisites

### 1. GCP CLI & kubectl 설정

```bash
# GCP 인증
gcloud auth login

# 프로젝트 설정
gcloud config set project finola-global

# kubectl 클러스터 연결
gcloud container clusters get-credentials ola-b2b --region=asia-northeast3 --project=finola-global
```

### 2. Docker 인증

```bash
gcloud auth configure-docker asia-northeast3-docker.pkg.dev
```

## Deployment Steps

### Step 1: Secret Manager 설정

환경변수를 GCP Secret Manager에 저장합니다.

```bash
# 시크릿 생성 (최초 1회)
gcloud secrets create ola-b2b-hana-securities-env-vars \
  --replication-policy="automatic" \
  --project=finola-global

# 환경변수 업로드
gcloud secrets versions add ola-b2b-hana-securities-env-vars \
  --data-file=.env \
  --project=finola-global
```

### Step 2: Namespace & Service Account 생성

```bash
# 환경변수 설정
PROJECT_ID=finola-global
KSA_NAME=hana-securities-app
KSA_NAMESPACE=hana-securities
GSA_NAME=ola-b2b-pod
GSA_EMAIL=${GSA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com

# Namespace 생성
kubectl apply -f k8s/namespace.yaml

# Kubernetes Service Account 생성
kubectl create -n ${KSA_NAMESPACE} serviceaccount ${KSA_NAME}

# IAM 정책 바인딩 (Workload Identity)
gcloud iam service-accounts add-iam-policy-binding $GSA_EMAIL \
  --role="roles/iam.workloadIdentityUser" \
  --member="serviceAccount:${PROJECT_ID}.svc.id.goog[${KSA_NAMESPACE}/${KSA_NAME}]" \
  --project=${PROJECT_ID}

# KSA에 GCP SA annotation 추가
kubectl annotate serviceaccount $KSA_NAME \
  --namespace $KSA_NAMESPACE \
  iam.gke.io/gcp-service-account=$GSA_EMAIL \
  --overwrite
```

### Step 3: Docker Image Build & Push

```bash
# 빌드 (linux/amd64 플랫폼)
docker build --platform linux/amd64 \
  -t asia-northeast3-docker.pkg.dev/finola-global/ola-b2b/hana-news-api:latest .

# 푸시
docker push asia-northeast3-docker.pkg.dev/finola-global/ola-b2b/hana-news-api:latest
```

### Step 4: K8S 리소스 배포

```bash
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/ingress.yaml
```

### Step 5: 배포 확인

```bash
# Pod 상태 확인
kubectl get pods -n hana-securities

# 로그 확인
kubectl logs -n hana-securities -l app=hana-news-api

# Ingress 상태 확인
kubectl get ingress -n hana-securities

# 인증서 상태 확인
kubectl get managedcertificate -n hana-securities
```

## Domain & SSL Setup

### DNS 설정 (AWS Route53)

Route53 호스팅 영역 `onelineai.com`에 다음 레코드를 추가합니다:

| 레코드 이름 | 유형 | 값 |
|------------|------|-----|
| `api.hana-securities` | A | `136.110.137.91` |
| `_acme-challenge.api.hana-securities` | CNAME | GCP DNS Authorization 값 |

### GCP Static IP 생성

```bash
gcloud compute addresses create api-hana-securities-static-ip-2501290930 \
  --global \
  --ip-version=IPV4 \
  --project=finola-global

# IP 확인
gcloud compute addresses describe api-hana-securities-static-ip-2501290930 \
  --global --project=finola-global --format="get(address)"
```

### DNS Authorization 생성

```bash
gcloud certificate-manager dns-authorizations create hana-securities-dns-auth \
  --domain="api.hana-securities.onelineai.com" \
  --project=finola-global

# CNAME 레코드 정보 확인
gcloud certificate-manager dns-authorizations describe hana-securities-dns-auth \
  --project=finola-global --format="yaml(dnsResourceRecord)"
```

## Environment Variables

환경변수는 Secret Manager에서 관리되며, initContainer가 Pod 시작 시 자동으로 가져옵니다.

| 변수 | 설명 |
|------|------|
| `SILVER_DB_HOST` | Silver DB 호스트 (tunnel.onelineai.com) |
| `SILVER_DB_PORT` | Silver DB 포트 (5432) |
| `SILVER_DB_NAME` | Silver DB 이름 (etl) |
| `SILVER_DB_USER` | Silver DB 사용자 |
| `SILVER_DB_PASSWORD` | Silver DB 비밀번호 |
| `SILVER_DB_SCHEMA` | Silver 스키마 (silver) |
| `GOLD_DB_HOST` | Gold DB 호스트 (10.35.64.2 - Private IP) |
| `GOLD_DB_PORT` | Gold DB 포트 (5432) |
| `GOLD_DB_NAME` | Gold DB 이름 (hana_securities) |
| `GOLD_DB_USER` | Gold DB 사용자 |
| `GOLD_DB_PASSWORD` | Gold DB 비밀번호 |
| `GOLD_DB_SCHEMA` | Gold 스키마 (gold) |
| `SERVER_PORT` | 서버 포트 (8080) |
| `BATCH_INTERVAL_MINUTES` | 배치 주기 (10분) |
| `LOG_LEVEL` | 로그 레벨 (info) |

### 환경변수 업데이트

```bash
# .env 파일 수정 후
gcloud secrets versions add ola-b2b-hana-securities-env-vars \
  --data-file=.env \
  --project=finola-global

# Pod 재시작 (새 환경변수 적용)
kubectl delete pod -n hana-securities -l app=hana-news-api
```

## Database Connectivity

| DB | 접근 방식 | 이유 |
|----|----------|------|
| Silver (onelineai-etl) | `tunnel.onelineai.com` | 다른 VPC - 터널 경유 |
| Gold (ola-b2b) | `10.35.64.2` (Private IP) | 같은 VPC - 직접 연결 |

## Monitoring & Troubleshooting

### Pod 상태 확인

```bash
# Pod 목록
kubectl get pods -n hana-securities

# Pod 상세 정보
kubectl describe pod -n hana-securities -l app=hana-news-api

# 실시간 로그
kubectl logs -n hana-securities -l app=hana-news-api -f
```

### Backend Health 확인

```bash
gcloud compute backend-services get-health \
  k8s1-54fa8892-hana-securities-hana-news-api-8080-5f064426 \
  --global --project=finola-global
```

### 일반적인 문제 해결

1. **Pod CrashLoopBackOff**
   - 로그 확인: `kubectl logs -n hana-securities -l app=hana-news-api --all-containers`
   - DB 연결 문제일 가능성 높음

2. **Ingress ADDRESS 없음**
   - `ingressClassName` 제거 확인
   - 이벤트 확인: `kubectl describe ingress -n hana-securities`

3. **인증서 Provisioning 지연**
   - DNS 레코드 확인: `dig +short api.hana-securities.onelineai.com`
   - 보통 10-30분 소요

## API Endpoints

| Method | Endpoint | 설명 |
|--------|----------|------|
| GET | `/health` | 헬스체크 |
| GET | `/docs` | Swagger UI |
| GET | `/v1/news` | 뉴스 목록 조회 |
| GET | `/v1/news/:id` | 뉴스 상세 조회 |

## Rolling Update

새 버전 배포 시:

```bash
# 1. 새 이미지 빌드 & 푸시
docker build --platform linux/amd64 \
  -t asia-northeast3-docker.pkg.dev/finola-global/ola-b2b/hana-news-api:v1.1.0 .
docker push asia-northeast3-docker.pkg.dev/finola-global/ola-b2b/hana-news-api:v1.1.0

# 2. Deployment 이미지 업데이트
kubectl set image deployment/hana-news-api \
  hana-news-api=asia-northeast3-docker.pkg.dev/finola-global/ola-b2b/hana-news-api:v1.1.0 \
  -n hana-securities

# 3. 롤아웃 상태 확인
kubectl rollout status deployment/hana-news-api -n hana-securities
```
