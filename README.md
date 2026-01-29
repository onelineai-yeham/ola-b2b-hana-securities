# Hana Securities News API

하나증권 뉴스 API 서버 - Silver 스키마에서 Gold 스키마로 번역된 뉴스를 동기화하고 API로 제공합니다.

## 기능

- **ETL 배치 작업**: 10분 주기로 Silver → Gold 데이터 동기화
- **뉴스 API**: 뉴스 목록/상세 조회, 티커 기반 필터링

## 프로젝트 구조

```
.
├── cmd/server/          # 애플리케이션 진입점
├── internal/
│   ├── config/          # 환경설정
│   ├── db/              # DB 연결
│   ├── model/           # 도메인 모델
│   ├── repository/      # 데이터 접근 계층
│   ├── service/         # 비즈니스 로직
│   ├── handler/         # HTTP 핸들러
│   └── scheduler/       # 배치 스케줄러
├── migrations/          # DB 마이그레이션
├── k8s/                 # Kubernetes 매니페스트
├── Dockerfile
└── .env.example
```

## 시작하기

### 1. 환경 설정

```bash
cp .env.example .env
# .env 파일의 DB 비밀번호 수정
```

### 2. Gold 스키마 생성

```bash
# ola-b2b DB에 접속하여 마이그레이션 실행
psql -h <gold-db-host> -U <user> -d hana_securities -f migrations/001_create_gold_tables.sql
```

### 3. 로컬 실행

```bash
go run ./cmd/server
```

### 4. 빌드

```bash
go build -o hana-news-api ./cmd/server
```

## API 문서

Swagger UI를 통해 API 문서를 확인할 수 있습니다:
- **로컬**: http://localhost:8080/docs
- **운영**: https://hana-news-api.ola-b2b.onelineai.com/docs

## API 엔드포인트

| Method | Endpoint | 설명 |
|--------|----------|------|
| GET | `/health` | 헬스체크 |
| GET | `/docs` | Swagger UI (API 문서) |
| GET | `/v1/news` | 뉴스 목록 조회 |
| GET | `/v1/news/:id` | 뉴스 상세 조회 |

### GET /v1/news 쿼리 파라미터

| 파라미터 | 타입 | 설명 |
|---------|------|------|
| source | string | `jp_minkabu` 또는 `cn_wind` |
| ticker | string | 티커 코드로 필터링 |
| from | RFC3339 | 시작 시간 |
| to | RFC3339 | 종료 시간 |
| page | int | 페이지 번호 (기본: 1) |
| limit | int | 페이지 크기 (기본: 20, 최대: 100) |

### 응답 예시

```json
{
  "data": [
    {
      "id": "jp_minkabu_12345",
      "source": "jp_minkabu",
      "headline": "번역된 헤드라인",
      "tickers": ["7203", "9984"],
      "creation_time": "2026-01-29T09:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 1400
  }
}
```

## 배포

### Docker 빌드

```bash
docker build -t hana-news-api:latest .
```

### Kubernetes 배포

```bash
# ConfigMap과 Secret 적용
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/secret.yaml

# 배포
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

## 환경 변수

| 변수 | 설명 | 기본값 |
|------|------|--------|
| SERVER_PORT | HTTP 서버 포트 | 8080 |
| BATCH_INTERVAL_MINUTES | 배치 주기 (분) | 10 |
| LOG_LEVEL | 로그 레벨 | info |
| SILVER_DB_* | Silver DB 연결 정보 | - |
| GOLD_DB_* | Gold DB 연결 정보 | - |

## 데이터 소스

- **JP Minkabu**: `silver.jp_minkabu_translated_news` → `gold.jp_minkabu_translated_news`
- **CN Wind**: `silver.cn_wind_translated_news` → `gold.cn_wind_translated_news`
