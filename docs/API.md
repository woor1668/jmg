# API Reference

모든 API는 `Authorization: Bearer <token>` 헤더 필요 (이미지 서빙 제외).

## 이미지

| 메서드 | 경로 | 설명 |
|---|---|---|
| `POST` | `/api/upload` | 이미지 업로드 (multipart, `file` + 선택 `folder`) |
| `PATCH` | `/api/images` | 이미지 이름/폴더 변경 |
| `DELETE` | `/api/images?id=` | 이미지 삭제 |
| `POST` | `/api/bulk-move` | 벌크 이동 `{ids, folder}` |
| `POST` | `/api/bulk-delete` | 벌크 삭제 `{ids}` |
| `GET` | `/:folder/:slug` | 이미지 서빙 (인증 불필요) |

## 폴더

| 메서드 | 경로 | 설명 |
|---|---|---|
| `GET` | `/api/browse/:folder` | 폴더 탐색 (하위폴더 + 이미지 목록) |
| `GET` | `/api/folders` | 전체 폴더 목록 + 이미지 수 |
| `POST` | `/api/folder-create` | 폴더 생성 `{path}` |
| `POST` | `/api/folder-rename` | 폴더 이름변경 `{old_name, new_name}` |
| `POST` | `/api/folder-delete` | 폴더 삭제 (하위 포함) `{path}` |

## 기타

| 메서드 | 경로 | 설명 |
|---|---|---|
| `GET` | `/api/health` | 헬스체크 |
| `GET` | `/admin/` | 관리자 웹 UI |
