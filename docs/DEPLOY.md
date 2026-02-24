# 배포 가이드

## 방법 1: Docker Hub에서 내려받기 (권장)

이미 빌드된 이미지를 바로 사용.

```bash
# 1. 작업 디렉토리 생성
mkdir jmg && cd jmg

# 2. 설정 파일 생성
cat > config.yaml << 'EOF'
auth:
  token: "여기에-토큰-입력"
server:
  host: "0.0.0.0"
  port: 8080
storage:
  data_dir: "/app/data"
EOF

# 3. 실행
docker run -d \
  --name jmg \
  --restart unless-stopped \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  your-dockerhub-user/jmg:latest

# 또는 환경변수로 토큰 지정 (config.yaml 없이)
docker run -d \
  --name jmg \
  --restart unless-stopped \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -e AUTH_TOKEN="여기에-토큰-입력" \
  your-dockerhub-user/jmg:latest
```

## 방법 2: GitHub에서 소스 빌드

```bash
# 1. 클론
git clone https://github.com/your-user/jmg.git
cd jmg

# 2. 설정
cp config.example.yaml config.yaml
# config.yaml에서 토큰 수정

# 3. 빌드 & 실행
docker compose up -d
```

## 방법 3: 직접 빌드 (Docker 없이)

```bash
git clone https://github.com/your-user/jmg.git
cd jmg
go build -o jmg .
cp config.example.yaml config.yaml
./jmg --config config.yaml
```

---

## Docker 이미지 빌드 & 푸시

직접 Docker Hub에 올리려면:

```bash
# 빌드
docker build -t your-dockerhub-user/jmg:latest .

# 멀티 아키텍처 (amd64 + arm64)
docker buildx build --platform linux/amd64,linux/arm64 \
  -t your-dockerhub-user/jmg:latest --push .

# 또는 단일 아키텍처 푸시
docker push your-dockerhub-user/jmg:latest
```

---

## 설정

`config.yaml` 또는 환경변수로 설정:

| 환경변수 | 설명 | 기본값 |
|---|---|---|
| `AUTH_TOKEN` | API 인증 토큰 (필수) | — |
| `BASE_URL` | 외부 접근 URL | `http://host:port` |
| `PORT` | 서버 포트 | `8080` |
| `DATA_DIR` | 데이터 디렉토리 | `./data` |

## 관리

```bash
# 로그 확인
docker logs jmg

# 중지 / 시작 / 재시작
docker stop jmg
docker start jmg
docker restart jmg

# 삭제 (데이터는 ./data에 보존)
docker rm -f jmg

# 업데이트 (Docker Hub)
docker pull your-dockerhub-user/jmg:latest
docker rm -f jmg
# 위의 docker run 명령 다시 실행

# 업데이트 (소스 빌드)
git pull
docker compose up -d --build
```

## 데이터 구조

```
data/
├── images.db        # SQLite DB
├── images/          # 원본 이미지 (sharded)
│   ├── ab/
│   │   └── abcdef_photo.png
│   └── ...
└── thumbnails/      # 자동 생성 썸네일
```

`data/` 디렉토리만 백업하면 전체 복원 가능.

## 백업 & 복원

```bash
# 백업
tar czf jmg-backup-$(date +%Y%m%d).tar.gz data/

# 복원
tar xzf jmg-backup-20260224.tar.gz
docker compose up -d
```
