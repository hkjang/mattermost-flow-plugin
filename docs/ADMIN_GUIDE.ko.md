# 관리자 가이드

이 문서는 Flow 플러그인을 설치, 활성화, 운영하는 Mattermost 관리자와 보드 소유자를 위한 가이드입니다.

## 요구 사항

- 플러그인 매니페스트 최소 버전 기준 Mattermost 서버 `6.2.1` 이상
- Mattermost 서버에서 플러그인 업로드가 활성화되어 있을 것
- 플러그인을 설치하고 활성화할 수 있는 권한

## 플러그인 설치

### 옵션 1: 릴리즈 번들 업로드

1. [GitHub Releases](https://github.com/hkjang/mattermost-flow-plugin/releases)에서 `com.mattermost.flow-plugin-<version>.tar.gz`를 다운로드합니다.
2. `System Console -> Plugin Management`를 엽니다.
3. `.tar.gz` 번들을 업로드합니다.
4. 업로드 후 플러그인을 활성화합니다.

### 옵션 2: `mmctl`로 설치

```bash
mmctl plugin add dist/com.mattermost.flow-plugin-<version>.tar.gz --local
mmctl plugin enable com.mattermost.flow-plugin
```

## 플러그인 업로드 활성화

플러그인 업로드가 비활성화되어 있다면 Mattermost 설정에서 켭니다.

```json
{
  "PluginSettings": {
    "EnableUploads": true
  }
}
```

환경에 따라 설정 적용 후 Mattermost 재시작이 필요할 수 있습니다.

## 권한 모델

Flow는 범위 접근 제어에 Mattermost 멤버십과 관리자 권한을 그대로 사용합니다.

- 보드 조회 사용자는 해당 보드 범위의 멤버여야 합니다
- 보드 관리자는 보드 설정과 구조를 수정할 수 있습니다
- 팀 관리자는 팀 안의 보드를 관리할 수 있습니다
- 시스템 관리자는 모든 보드를 관리할 수 있습니다

보드 단위 관리자는 보드 메타데이터에 함께 저장됩니다.

## 운영 동작

### 저장소

Flow는 운영 데이터를 Mattermost 플러그인 KV 저장소에 보관합니다.

- 보드 메타데이터와 컬럼
- 카드와 의존성
- 활동 이력
- 사용자 보기 설정
- 채널 기본 보드 매핑
- 마감 임박 알림 상태

현재 플러그인 설계에서는 외부 데이터베이스가 필요하지 않습니다.

### 알림

보드 설정이 허용하면 Flow는 연결된 채널에 업데이트를 포스트할 수 있습니다.

사용 가능한 보드 단위 설정 예시:

- `post_updates`
- `post_due_soon`
- `allow_mentions`
- `default_view`

마감 임박 스캔은 매시간 실행되는 백그라운드 클러스터 잡으로 처리됩니다.

### 릴리즈 번들의 실행 권한

릴리즈 아카이브는 `server/dist/` 아래 파일이 실행 권한 `0755`로 저장되도록 패키징됩니다. 이 설정은 Mattermost가 플러그인 번들을 압축 해제한 뒤 Linux나 macOS 호스트에서 서버 바이너리를 실행하지 못하는 흔한 문제를 막아 줍니다.

## 권장 배포 순서

1. 스테이징 Mattermost 인스턴스에 플러그인을 설치합니다.
2. 팀 보드 하나와 채널 보드 하나를 생성합니다.
3. 보드 뷰, 간트 뷰, 슬래시 명령, 채널 포스트를 확인합니다.
4. 운영 환경에서 플러그인을 활성화합니다.
5. 팀 소유자에게 사용자 가이드와 보드 운영 규칙을 공유합니다.

## 업그레이드와 롤백

- 업그레이드: 더 새로운 `.tar.gz` 번들을 업로드하거나 `mmctl`로 새 릴리즈를 설치합니다
- 롤백: 이전 릴리즈 번들을 다시 업로드하고 필요하면 재활성화합니다

Flow는 KV 저장소에 데이터를 보관하므로, 뒤로 롤백할 때는 플러그인 버전 차이가 너무 크지 않도록 유지하는 편이 안전합니다.

## 문제 해결

### 플러그인 업로드 실패

- `PluginSettings.EnableUploads`가 활성화되어 있는지 확인하세요
- 업로드한 파일이 저장소 소스 압축본이 아니라 생성된 `.tar.gz` 번들인지 확인하세요

### 사용자가 `Not authorized`를 받는 경우

- 사용자가 Mattermost에 로그인한 상태인지 확인하세요
- 사용자가 해당 보드의 팀 또는 채널 멤버인지 확인하세요
- 리버스 프록시가 플러그인 라우트의 Mattermost 인증 헤더를 제거하지 않는지 확인하세요

### 마감 임박 포스트가 올라오지 않는 경우

- 보드가 채널 범위인지 확인하세요
- 보드 설정에서 `post_due_soon`이 활성화되어 있는지 확인하세요
- 카드에 마감일이 있고 이미 완료 상태가 아닌지 확인하세요

### 빠른 액션에서 담당자 멘션이 되지 않는 경우

- 보드 설정에서 `allow_mentions`가 활성화되어 있는지 확인하세요

## 관련 문서

- [README](../README.md)
- [사용자 가이드](./USER_GUIDE.ko.md)
- [개발 가이드](./DEVELOPMENT_GUIDE.ko.md)
- [릴리즈 가이드](./RELEASE_GUIDE.ko.md)
