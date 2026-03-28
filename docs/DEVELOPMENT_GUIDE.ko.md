# 개발 가이드

이 문서는 Flow 플러그인 코드베이스에 기여하는 개발자를 위한 가이드입니다.

## 스택

- 서버 플러그인: Go
- 웹앱 플러그인: React + TypeScript
- 라우터/API: `gorilla/mux`
- 저장소: Mattermost 플러그인 KV 저장소
- 실시간 업데이트: server-sent events와 로컬 클라이언트 sync bridge

## 저장소 구조

```text
assets/                  릴리즈에 포함되는 플러그인 에셋
build/                   매니페스트, 패키징, 플러그인 제어 헬퍼
docs/                    사용자, 관리자, 개발, 릴리즈 문서
server/                  Go 플러그인 서버 코드
server/command/          슬래시 명령 처리기
server/store/            KV 저장소 래퍼
webapp/src/              React UI, Mattermost 연동, 스타일, 테스트
plugin.json              Mattermost 플러그인 매니페스트
Makefile                 로컬 빌드, 테스트, 배포, 릴리즈 헬퍼
```

## 주요 런타임 진입점

### 서버

- [server/plugin.go](../server/plugin.go): 플러그인 활성화, 봇 설정, 슬래시 명령 등록, 백그라운드 잡 스케줄링
- [server/api.go](../server/api.go): `/api/v1` 아래 플러그인 전용 API
- [server/service.go](../server/service.go): 보드, 카드, 활동, 간트, 요약에 대한 비즈니스 로직
- [server/store.go](../server/store.go): KV 저장소 계약과 키 처리
- [server/event_broker.go](../server/event_broker.go): SSE 구독 브로커

### 웹앱

- [webapp/src/index.tsx](../webapp/src/index.tsx): Mattermost 플러그인 등록
- [webapp/src/flow_page.tsx](../webapp/src/flow_page.tsx): 메인 보드 및 간트 경험
- [webapp/src/flow_post.tsx](../webapp/src/flow_post.tsx): 커스텀 포스트 렌더링과 빠른 액션
- [webapp/src/client.ts](../webapp/src/client.ts): 플러그인 API 클라이언트
- [webapp/src/flow_sync.ts](../webapp/src/flow_sync.ts): 같은 탭/다른 탭 동기화 브리지

## 로컬 설정

사전 준비:

- Go `1.25+`
- Node.js `24.13.1`
- 플러그인 업로드가 활성화된 Mattermost 인스턴스

설치와 빌드:

```bash
make
```

유용한 타깃:

```bash
make test
make dist
make deploy
make watch
make logs
```

## API 표면

플러그인 서버 API는 다음 경로 아래에 노출됩니다.

```text
/plugins/com.mattermost.flow-plugin/api/v1
```

주요 엔드포인트:

- `GET /boards`
- `POST /boards`
- `GET /boards/{id}`
- `PATCH /boards/{id}`
- `DELETE /boards/{id}`
- `GET /boards/{id}/stream`
- `GET /boards/summary/stream`
- `GET /boards/{id}/cards`
- `GET /boards/{id}/gantt`
- `GET /boards/{id}/activity`
- `PUT /boards/{id}/preferences`
- `GET /boards/{id}/users`
- `POST /cards`
- `PATCH /cards/{id}`
- `POST /cards/{id}/move`
- `POST /cards/{id}/actions/{action}`
- `POST /cards/{id}/dependencies`
- `POST /cards/{id}/comments`

요청은 Mattermost 인증 헤더와 보드 범위 권한 검사를 전제로 동작합니다.

## 데이터 모델

핵심 엔터티:

- `Board`
- `BoardColumn`
- `Card`
- `Dependency`
- `Activity`
- `Preference`
- `DueSoonNotification`

데이터는 Mattermost 플러그인 KV 저장소에 저장됩니다. 보드 요약, 활동, 실시간 업데이트는 write path 가까이에 유지해서 사이드바와 보드 화면이 전체 새로고침 없이도 빠르게 자신을 patch할 수 있게 합니다.

## 협업 기능에서 주의할 점

- Flow는 보드 업데이트와 마감 임박 알림에 Mattermost 포스트를 사용합니다
- 포스트 빠른 액션은 플러그인 API를 통해 카드를 직접 변경합니다
- SSE 업데이트는 열려 있는 보드와 보드 목록을 사용자 간 동기화합니다
- 백그라운드 클러스터 잡이 매시간 마감 임박 카드를 스캔합니다

mutation 흐름을 바꿀 때는 서버의 이벤트 발행 경로와 클라이언트의 patching 로직을 함께 업데이트해야 합니다.

## 테스트

푸시 전 권장 점검:

```bash
go test ./server/... ./server/command ./server/store/...
cd webapp && npm run check-types
cd webapp && npm run test
```

배포 번들까지 포함한 전체 검증:

```bash
make dist
```

## 문서 업데이트 원칙

사용자에게 보이는 동작을 바꿨다면 다음 문서를 함께 업데이트하세요.

- [README](../README.md)
- [사용자 가이드](./USER_GUIDE.ko.md)
- [관리자 가이드](./ADMIN_GUIDE.ko.md)
- 패키징이나 릴리즈 동작이 바뀌었다면 [릴리즈 가이드](./RELEASE_GUIDE.ko.md)
