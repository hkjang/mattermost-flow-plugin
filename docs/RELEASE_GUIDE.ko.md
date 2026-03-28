# 릴리즈 가이드

이 문서는 Mattermost Flow 플러그인 릴리즈를 빌드, 태깅, 배포, 검증하는 방법을 설명합니다.

## 버전 규칙

플러그인 버전은 Git 태그를 기반으로 빌드 시점에 주입됩니다.

- 현재 커밋에 `v0.1.2` 같은 태그가 있으면 번들 버전은 `0.1.2`가 됩니다
- 커밋에 태그가 없으면 가장 가까운 태그와 현재 커밋 해시를 조합해 버전을 만듭니다

따라서 릴리즈 빌드는 깨끗하고 의도적인 Git 태그에서 만드는 것이 좋습니다.

## 로컬 빌드

로컬에서 릴리즈 번들을 생성합니다.

```bash
make dist
```

출력:

```text
dist/com.mattermost.flow-plugin-<version>.tar.gz
```

## 태그 기반 GitHub 릴리즈 흐름

이 저장소는 GitHub Actions로 릴리즈를 자동 배포합니다.

1. 최신 커밋을 `main`에 푸시합니다
2. Annotated release tag를 생성합니다
3. 태그를 GitHub로 푸시합니다
4. GitHub Actions가 플러그인 번들을 빌드하고 `SHA256SUMS.txt`를 생성한 뒤 두 파일을 GitHub Release에 업로드합니다

예시:

```bash
git tag -a v0.1.2 -m "Mattermost Flow Plugin v0.1.2"
git push origin v0.1.2
```

## 헬퍼 타깃

`Makefile`에는 시맨틱 버전 태그를 위한 편의 타깃이 들어 있습니다.

```bash
make patch
make minor
make major
make patch-rc
make minor-rc
make major-rc
```

이 헬퍼는 `main` 또는 릴리즈 브랜치에서 실행하는 것을 전제로 하며, 생성된 태그를 원격으로 푸시합니다.

## 실행 권한

릴리즈 번들러는 `server/dist/` 아래 파일을 아카이브에 모드 `0755`로 기록합니다.

이 설정은 Mattermost 플러그인 번들이 압축 해제된 뒤에도 서버 바이너리가 실행 가능해야 하기 때문에 중요합니다. 커스텀 번들 단계가 이 실행 비트 유실 문제를 막아 주므로, 업로드는 성공했지만 플러그인이 시작되지 않는 상황을 피할 수 있습니다.

관련 구현 파일:

- [build/manifest/main.go](../build/manifest/main.go)
- [build/package_plugin.ps1](../build/package_plugin.ps1)
- [Makefile](../Makefile)

## 릴리즈 검증

태그를 푸시한 뒤 다음을 확인하세요.

1. GitHub Actions 릴리즈 워크플로우가 성공했는지 확인
2. GitHub Release가 생성되었는지 확인
3. 아래 두 자산이 모두 업로드되었는지 확인
   - `com.mattermost.flow-plugin-<version>.tar.gz`
   - `SHA256SUMS.txt`
4. 필요하면 로컬에서 SHA256 체크섬 검증

## 롤백

릴리즈에 문제가 있으면:

- Mattermost에서 플러그인을 비활성화합니다
- 이전 릴리즈 번들을 다시 설치합니다
- 플러그인을 다시 활성화합니다

플러그인은 상태를 KV 저장소에 보관하므로, 데이터 모델 변경이 포함된 릴리즈는 롤백 전에 스테이징에서 먼저 확인하는 편이 안전합니다.
