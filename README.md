# JMG

개인용 셀프 호스팅 이미지 호스팅 서비스.

이미지를 업로드하면 고유 URL이 생성되고, 블로그·메신저·문서 어디서든 바로 붙여넣어 쓸 수 있다. 파일탐색기 스타일의 관리 UI에서 폴더 정리, 이름 변경, 벌크 작업을 할 수 있고, 클립보드 붙여넣기와 드래그앤드롭 업로드를 지원한다.

## 주요 기능

- **즉시 공유** — 업로드 즉시 URL 생성, 클립보드 자동 복사
- **폴더 관리** — 하위폴더 지원, 파일탐색기 UI에서 드래그앤드롭·벌크 이동·삭제
- **중복 방지** — SHA-256 해시로 같은 이미지 중복 업로드 차단
- **썸네일 자동 생성** — 200/400/800px 리사이즈
- **Slug URL** — `example.com/photos/sunset` 형태의 깔끔한 URL
- **토큰 인증** — 업로드·관리는 토큰 필수, 이미지 열람은 공개
- **ETag 캐싱** — 브라우저 캐시 활용으로 빠른 서빙
- **단일 바이너리** — Go 빌드 하나로 끝, SQLite 내장 (CGO 불필요)
- **Docker 지원** — `docker compose up -d`

## 기술 스택

Go 1.22 · SQLite (modernc.org/sqlite) · 바닐라 HTML/CSS/JS

## 시작하기

```bash
cp config.example.yaml config.yaml   # 토큰 수정
docker compose up -d                 # 또는: go build -o jmg . && ./jmg --config config.yaml
```

`http://localhost:8080/admin/` 접속.

## 문서

- [API 레퍼런스](docs/API.md)
- [배포 가이드](docs/DEPLOY.md)
