# 배포 가이드

## Docker (권장)

```bash
cp config.example.yaml config.yaml
# config.yaml에서 토큰 수정

docker compose up -d
```

## 직접 빌드

```bash
go build -o jmg .
cp config.example.yaml config.yaml
./jmg --config config.yaml
```

## 설정

`config.yaml` 또는 환경변수로 설정:

| 환경변수 | 설명 | 기본값 |
|---|---|---|
| `AUTH_TOKEN` | API 인증 토큰 (필수) | — |
| `BASE_URL` | 외부 접근 URL | `http://host:port` |
| `PORT` | 서버 포트 | `8080` |
| `DATA_DIR` | 데이터 디렉토리 | `./data` |

## 데이터 구조

```
data/
├── images.db        # SQLite DB
├── images/          # 원본 이미지 (sharded)
│   ├── ab/
│   │   └── abcdef_photo.png
│   └── ...
└── thumbnails/      # 자동 생성 썸네일
    └── ...
```

`data/` 디렉토리만 백업하면 전체 복원 가능.
