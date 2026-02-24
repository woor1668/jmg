# 배포 가이드

## 이미지 빌드 & 푸시 (로컬에서 1회)

```bash
# GitHub Container Registry 로그인
docker login ghcr.io -u woor1668

# 빌드
docker build -t ghcr.io/woor1668/jmg:latest .

# 푸시
docker push ghcr.io/woor1668/jmg:latest
```

> 멀티 아키텍처 (amd64 + arm64) 빌드:
> ```bash
> docker buildx build --platform linux/amd64,linux/arm64 \
>   -t ghcr.io/woor1668/jmg:latest --push .
> ```

---

## 서버에 배포

### 방법 1: docker run (권장)

```bash
# 작업 디렉토리
mkdir jmg && cd jmg

# 설정 파일
cat > config.yaml << 'EOF'
auth:
  token: "여기에-토큰-입력"
server:
  host: "0.0.0.0"
  port: 8080
storage:
  data_dir: "/app/data"
EOF

# 실행
docker run -d \
  --name jmg \
  --restart unless-stopped \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  ghcr.io/woor1668/jmg:latest
```

환경변수로만 쓰려면 config.yaml 없이:
```bash
docker run -d \
  --name jmg \
  --restart unless-stopped \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -e AUTH_TOKEN="여기에-토큰-입력" \
  ghcr.io/woor1668/jmg:latest
```

### 방법 2: docker compose

```bash
git clone https://github.com/woor1668/jmg.git
cd jmg
cp config.example.yaml config.yaml   # 토큰 수정
docker compose up -d
```

### 방법 3: 직접 빌드 (Docker 없이)

```bash
git clone https://github.com/woor1668/jmg.git
cd jmg
go build -o jmg .
cp config.example.yaml config.yaml
./jmg --config config.yaml
```

---

## 관리

```bash
# 로그
docker logs jmg
docker logs -f jmg          # 실시간

# 중지 / 시작 / 재시작
docker stop jmg
docker start jmg
docker restart jmg

# 삭제 (데이터는 ./data에 보존)
docker rm -f jmg

# 업데이트
docker pull ghcr.io/woor1668/jmg:latest
docker rm -f jmg
# docker run 명령 다시 실행
```

## 설정

`config.yaml` 또는 환경변수:

| 환경변수 | 설명 | 기본값 |
|---|---|---|
| `AUTH_TOKEN` | API 인증 토큰 (필수) | — |
| `BASE_URL` | 외부 접근 URL | `http://host:port` |
| `PORT` | 서버 포트 | `8080` |
| `DATA_DIR` | 데이터 디렉토리 | `./data` |

## 백업 & 복원

```bash
# 백업
tar czf jmg-backup-$(date +%Y%m%d).tar.gz data/

# 복원
tar xzf jmg-backup-20260224.tar.gz
```

`data/` 디렉토리만 백업하면 전체 복원 가능.
