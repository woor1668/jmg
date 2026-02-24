# JMG — 개인 이미지 호스팅

셀프 호스팅 이미지 업로더. 토큰 인증, 폴더 정리, 썸네일 자동 생성, 중복 방지(SHA-256).

## 빠른 시작

### Docker
```bash
cp config.example.yaml config.yaml   # 토큰 수정
docker compose up -d
```

### 직접 빌드
```bash
go build -o jmg .
cp config.example.yaml config.yaml   # 토큰 수정
./jmg --config config.yaml
```

`http://localhost:8080/admin/` 으로 접속.

## 설정

`config.yaml` 또는 환경변수로 설정 가능:

| 환경변수 | 설명 | 기본값 |
|---|---|---|
| `AUTH_TOKEN` | API 인증 토큰 (필수) | — |
| `BASE_URL` | 외부 접근 URL | `http://host:port` |
| `PORT` | 서버 포트 | `8080` |
| `DATA_DIR` | 데이터 디렉토리 | `./data` |

## API

| 메서드 | 경로 | 설명 |
|---|---|---|
| `POST` | `/api/upload` | 이미지 업로드 |
| `GET` | `/api/browse/:folder` | 폴더 탐색 |
| `POST` | `/api/folder-create` | 폴더 생성 |
| `POST` | `/api/folder-rename` | 폴더 이름변경 |
| `POST` | `/api/folder-delete` | 폴더 삭제 |
| `PATCH` | `/api/images` | 이미지 이름/폴더 변경 |
| `DELETE` | `/api/images?id=` | 이미지 삭제 |
| `POST` | `/api/bulk-move` | 벌크 이동 |
| `POST` | `/api/bulk-delete` | 벌크 삭제 |
| `GET` | `/:folder/:slug` | 이미지 서빙 |

모든 API는 `Authorization: Bearer <token>` 헤더 필요 (이미지 서빙 제외).
