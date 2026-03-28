const revealTargets = Array.from(document.querySelectorAll('.reveal'));
const yearNode = document.getElementById('year');

const translations = {
    en: {
        'meta.title': 'Mattermost Flow Plugin',
        'meta.description': 'Mattermost Flow Plugin adds kanban boards, gantt timelines, live collaboration, and deployment-friendly plugin packaging directly inside Mattermost.',
        'meta.ogTitle': 'Mattermost Flow Plugin',
        'meta.ogDescription': 'Kanban boards, gantt timelines, real-time sync, and Mattermost-native collaboration in one plugin.',
        'brand.homeAria': 'Mattermost Flow Plugin home',
        'brand.name': 'Mattermost Flow',
        'brand.tagline': 'Kanban + Gantt for Mattermost',
        'nav.features': 'Features',
        'nav.operators': 'Operators',
        'nav.docs': 'Docs',
        'nav.install': 'Install',
        'button.github': 'GitHub',
        'button.latestRelease': 'Latest release',
        'button.download': 'Download plugin',
        'button.exploreGuides': 'Explore guides',
        'button.downloadLatest': 'Download latest release',
        'button.viewRepo': 'View repository',
        'hero.eyebrow': 'Mattermost plugin for delivery teams',
        'hero.title': 'Turn channel momentum into delivery flow.',
        'hero.lede': 'Mattermost Flow Plugin brings kanban boards, gantt timelines, live updates, deep links, and quick post actions into the same workspace where your team already chats.',
        'hero.ledeAccent': 'Keep work planning and delivery timelines inside the same channel context.',
        'stats.syncedViews': 'synced views',
        'stats.noDatabase': 'extra database required',
        'stats.executableModes': 'safe executable bundle modes',
        'showcase.channel': '#release-war-room',
        'lane.todo': 'Todo',
        'lane.inProgress': 'In Progress',
        'lane.done': 'Done',
        'card.shipDocs': 'Ship v0.1.2 docs refresh',
        'card.dueToday': 'Due today',
        'card.high': 'High',
        'card.prepareNotes': 'Prepare release notes',
        'card.twoAssignees': '2 assignees',
        'card.docs': 'Docs',
        'card.buildBundle': 'Build plugin bundle',
        'card.liveUpdates': 'Live updates',
        'card.reviewPostActions': 'Review post actions',
        'card.pushTag': 'Push release tag',
        'card.githubActions': 'GitHub Actions',
        'gantt.title': 'Gantt timeline',
        'gantt.view': 'Week view',
        'gantt.kickoff': 'Kickoff',
        'gantt.build': 'Build',
        'gantt.release': 'Release',
        'post.eyebrow': 'Flow post action',
        'post.title': 'Due soon: Ship v0.1.2 docs refresh',
        'post.assignMe': 'Assign to me',
        'post.pushDay': 'Push +1 day',
        'post.openGantt': 'Open gantt',
        'strip.text': 'Built on the Mattermost plugin model: custom API, KV storage, slash commands, live SSE streams, post interactions, and GitHub-ready releases.',
        'features.eyebrow': 'Why teams use it',
        'features.title': 'One workspace for planning, execution, and channel feedback.',
        'feature.1.title': 'Kanban and gantt stay in sync',
        'feature.1.desc': 'Cards, dates, progress, milestones, and dependencies power both views, so teams do not maintain separate planning tools.',
        'feature.2.title': 'Mattermost-native collaboration',
        'feature.2.desc': 'Use channel posts, mentions, slash commands, deep links, and quick post actions without sending people to another app.',
        'feature.3.title': 'Real-time board awareness',
        'feature.3.desc': 'SSE updates and local sync keep boards, cards, gantt views, and sidebar summaries fresh across open sessions.',
        'feature.4.title': 'Operator-friendly deployment',
        'feature.4.desc': 'Ship a single `.tar.gz` plugin bundle with executable-safe server binaries and tag-driven GitHub release automation.',
        'inside.eyebrow': 'Inside the flow',
        'inside.title': 'Start in chat, land on the exact card, finish with one click.',
        'inside.item1': 'A channel post announces a card movement or due-soon reminder.',
        'inside.item2': 'A teammate uses `Assign to me`, `Push +1 day`, or `Mark done` directly from the post.',
        'inside.item3': 'Open boards and sidebar summaries update without a full refresh.',
        'signal.deepLinks': 'Deep links',
        'signal.deepLinksDesc': 'Board, gantt, and card URLs',
        'signal.scopeAware': 'Scope-aware assignees',
        'signal.scopeAwareDesc': 'Team and channel members only',
        'signal.deliveryHygiene': 'Delivery hygiene',
        'signal.deliveryHygieneDesc': 'Checklist, dependencies, activity history',
        'operators.eyebrow': 'For operators',
        'operators.title': 'Designed to fit Mattermost plugin operations, not fight them.',
        'ops.1.title': 'KV-store first',
        'ops.1.desc': 'Boards, cards, activity, preferences, default board mappings, and due-soon state live in the Mattermost plugin KV store.',
        'ops.2.title': 'Permission-aware',
        'ops.2.desc': 'Board access respects team, channel, system admin, and board admin boundaries instead of inventing a second auth system.',
        'ops.3.title': 'Release-safe packaging',
        'ops.3.desc': 'Server binaries are bundled with executable mode `0755`, avoiding the classic plugin-start failure after extraction.',
        'docs.eyebrow': 'Docs',
        'docs.title': 'Pick the guide that matches your role.',
        'doc.user.title': 'User Guide',
        'doc.user.desc': 'How to work with boards, cards, gantt, and post actions',
        'doc.admin.title': 'Admin Guide',
        'doc.admin.desc': 'Install, operate, troubleshoot, and roll out the plugin safely',
        'doc.dev.title': 'Development Guide',
        'doc.dev.desc': 'Understand the API surface, KV model, SSE flows, and local workflows',
        'doc.release.title': 'Release Guide',
        'doc.release.desc': 'Build, tag, publish, verify, and roll back releases with confidence',
        'install.eyebrow': 'Install',
        'install.title': 'Build locally or upload the latest release.',
        'install.desc': 'Mattermost Flow ships as a standard plugin bundle, so teams can install it through Mattermost Plugin Management or distribute it through internal plugin channels.',
        'cta.eyebrow': 'Ready to try it?',
        'cta.title': 'Bring structured delivery flow into the same place your team already talks.',
        'footer.name': 'Mattermost Flow Plugin',
        'footer.tagline': 'Built for channel-native planning, execution, and release coordination.',
    },
    ko: {
        'meta.title': 'Mattermost Flow Plugin',
        'meta.description': 'Mattermost 안에서 칸반 보드, 간트 타임라인, 실시간 협업, 배포 친화적 플러그인 패키징을 함께 제공하는 플러그인입니다.',
        'meta.ogTitle': 'Mattermost Flow Plugin',
        'meta.ogDescription': '칸반 보드, 간트 타임라인, 실시간 동기화, Mattermost 네이티브 협업을 하나의 플러그인으로 제공합니다.',
        'brand.homeAria': 'Mattermost Flow Plugin 홈',
        'brand.name': 'Mattermost Flow',
        'brand.tagline': 'Mattermost용 칸반 + 간트',
        'nav.features': '기능',
        'nav.operators': '운영',
        'nav.docs': '문서',
        'nav.install': '설치',
        'button.github': 'GitHub',
        'button.latestRelease': '최신 릴리즈',
        'button.download': '플러그인 다운로드',
        'button.exploreGuides': '가이드 보기',
        'button.downloadLatest': '최신 릴리즈 다운로드',
        'button.viewRepo': '저장소 보기',
        'hero.eyebrow': '딜리버리 팀을 위한 Mattermost 플러그인',
        'hero.title': '채널의 흐름을 전달 가능한 작업 흐름으로 바꾸세요.',
        'hero.lede': 'Mattermost Flow Plugin은 칸반 보드, 간트 타임라인, 실시간 업데이트, 딥링크, 포스트 액션을 Mattermost 안으로 가져옵니다.',
        'hero.ledeAccent': '업무 계획과 일정 관리를 같은 채널 문맥 안에서 이어갈 수 있습니다.',
        'stats.syncedViews': '동기화된 보기',
        'stats.noDatabase': '추가 데이터베이스 불필요',
        'stats.executableModes': '안전한 실행 권한',
        'showcase.channel': '#release-war-room',
        'lane.todo': '할 일',
        'lane.inProgress': '진행 중',
        'lane.done': '완료',
        'card.shipDocs': 'v0.1.2 문서 업데이트 배포',
        'card.dueToday': '오늘 마감',
        'card.high': '높음',
        'card.prepareNotes': '릴리즈 노트 준비',
        'card.twoAssignees': '담당자 2명',
        'card.docs': '문서',
        'card.buildBundle': '플러그인 번들 빌드',
        'card.liveUpdates': '실시간 업데이트',
        'card.reviewPostActions': '포스트 액션 검토',
        'card.pushTag': '릴리즈 태그 푸시',
        'card.githubActions': 'GitHub Actions',
        'gantt.title': '간트 타임라인',
        'gantt.view': '주간 보기',
        'gantt.kickoff': '착수',
        'gantt.build': '빌드',
        'gantt.release': '릴리즈',
        'post.eyebrow': 'Flow 포스트 액션',
        'post.title': '마감 임박: v0.1.2 문서 업데이트 배포',
        'post.assignMe': '나에게 할당',
        'post.pushDay': '+1일 미루기',
        'post.openGantt': '간트 열기',
        'strip.text': 'Mattermost 플러그인 모델 위에 커스텀 API, KV 저장소, 슬래시 명령, 라이브 SSE 스트림, 포스트 인터랙션, GitHub 릴리즈 자동화까지 담았습니다.',
        'features.eyebrow': '팀이 쓰는 이유',
        'features.title': '계획, 실행, 채널 피드백을 하나의 작업 공간으로 묶습니다.',
        'feature.1.title': '칸반과 간트가 함께 움직입니다',
        'feature.1.desc': '카드, 날짜, 진행률, 마일스톤, 의존성이 두 화면을 함께 구동하므로 별도 계획 도구를 이중으로 관리할 필요가 없습니다.',
        'feature.2.title': 'Mattermost 네이티브 협업',
        'feature.2.desc': '채널 포스트, 멘션, 슬래시 명령, 딥링크, 빠른 포스트 액션을 다른 앱으로 이동하지 않고 그대로 활용할 수 있습니다.',
        'feature.3.title': '실시간 보드 가시성',
        'feature.3.desc': 'SSE와 로컬 동기화로 보드, 카드, 간트, 사이드바 요약이 열린 세션 사이에서도 빠르게 맞춰집니다.',
        'feature.4.title': '운영 친화적 배포',
        'feature.4.desc': '실행 권한이 보존된 서버 바이너리와 태그 기반 GitHub 릴리즈 자동화를 포함한 단일 `.tar.gz` 번들을 배포할 수 있습니다.',
        'inside.eyebrow': '흐름 안에서',
        'inside.title': '채팅에서 시작해 정확한 카드로 이동하고, 한 번의 클릭으로 마무리하세요.',
        'inside.item1': '채널 포스트가 카드 이동이나 마감 임박 상황을 바로 알려줍니다.',
        'inside.item2': '팀원은 포스트에서 바로 `나에게 할당`, `+1일 미루기`, `완료 처리`를 실행할 수 있습니다.',
        'inside.item3': '열려 있는 보드와 사이드바 요약은 전체 새로고침 없이 업데이트됩니다.',
        'signal.deepLinks': '딥링크',
        'signal.deepLinksDesc': '보드, 간트, 카드 URL',
        'signal.scopeAware': '범위 기반 담당자',
        'signal.scopeAwareDesc': '팀 또는 채널 멤버만 선택',
        'signal.deliveryHygiene': '운영 가시성',
        'signal.deliveryHygieneDesc': '체크리스트, 의존성, 활동 이력',
        'operators.eyebrow': '운영자 관점',
        'operators.title': 'Mattermost 플러그인 운영 방식에 자연스럽게 맞춥니다.',
        'ops.1.title': 'KV 저장 우선',
        'ops.1.desc': '보드, 카드, 활동, 환경설정, 기본 보드 매핑, 마감 임박 상태를 Mattermost 플러그인 KV 저장소에 보관합니다.',
        'ops.2.title': '권한 인지형 설계',
        'ops.2.desc': '팀, 채널, 시스템 관리자, 보드 관리자 경계를 그대로 따르며 별도 권한 체계를 새로 만들지 않습니다.',
        'ops.3.title': '릴리즈 안전성',
        'ops.3.desc': '서버 바이너리를 `0755` 실행 권한으로 묶어 압축 해제 후 플러그인이 실행되지 않는 고전적인 문제를 피합니다.',
        'docs.eyebrow': '문서',
        'docs.title': '역할에 맞는 가이드를 바로 고르세요.',
        'doc.user.title': '사용자 가이드',
        'doc.user.desc': '보드, 카드, 간트, 포스트 액션 사용법',
        'doc.admin.title': '관리자 가이드',
        'doc.admin.desc': '설치, 운영, 장애 대응, 롤아웃 방법',
        'doc.dev.title': '개발 가이드',
        'doc.dev.desc': 'API, KV 모델, SSE 흐름, 로컬 개발 워크플로우',
        'doc.release.title': '릴리즈 가이드',
        'doc.release.desc': '빌드, 태그, 배포, 검증, 롤백 절차',
        'install.eyebrow': '설치',
        'install.title': '직접 빌드하거나 최신 릴리즈를 바로 올리세요.',
        'install.desc': 'Mattermost Flow는 표준 플러그인 번들로 배포되므로 Plugin Management나 내부 배포 채널을 통해 쉽게 설치할 수 있습니다.',
        'cta.eyebrow': '시작할 준비가 됐나요?',
        'cta.title': '팀이 이미 대화하는 공간 안으로 구조화된 전달 흐름을 가져오세요.',
        'footer.name': 'Mattermost Flow Plugin',
        'footer.tagline': '채널 기반 계획, 실행, 릴리즈 조율을 위한 Mattermost Flow Plugin',
    },
};

function resolveLanguage() {
    const params = new URLSearchParams(window.location.search);
    const forced = (params.get('lang') || '').toLowerCase();
    if (forced === 'ko' || forced === 'en') {
        return forced;
    }

    const languages = Array.isArray(navigator.languages) && navigator.languages.length > 0 ? navigator.languages : [navigator.language || 'en'];
    return languages.some((value) => String(value).toLowerCase().startsWith('ko')) ? 'ko' : 'en';
}

function translatePage(language) {
    const dictionary = translations[language] || translations.en;
    const fallback = translations.en;

    document.documentElement.lang = language;
    document.documentElement.dataset.language = language;

    document.title = dictionary['meta.title'] || fallback['meta.title'];

    const metaDescription = document.getElementById('meta-description');
    if (metaDescription) {
        metaDescription.setAttribute('content', dictionary['meta.description'] || fallback['meta.description']);
    }

    const metaOgTitle = document.getElementById('meta-og-title');
    if (metaOgTitle) {
        metaOgTitle.setAttribute('content', dictionary['meta.ogTitle'] || fallback['meta.ogTitle']);
    }

    const metaOgDescription = document.getElementById('meta-og-description');
    if (metaOgDescription) {
        metaOgDescription.setAttribute('content', dictionary['meta.ogDescription'] || fallback['meta.ogDescription']);
    }

    const brandLink = document.getElementById('brand-link');
    if (brandLink) {
        brandLink.setAttribute('aria-label', dictionary['brand.homeAria'] || fallback['brand.homeAria']);
    }

    document.querySelectorAll('[data-i18n]').forEach((node) => {
        const key = node.getAttribute('data-i18n');
        if (!key) {
            return;
        }

        const value = dictionary[key] || fallback[key];
        if (typeof value === 'string') {
            node.textContent = value;
        }
    });
}

if (yearNode) {
    yearNode.textContent = String(new Date().getFullYear());
}

translatePage(resolveLanguage());

if ('IntersectionObserver' in window) {
    const observer = new IntersectionObserver((entries, currentObserver) => {
        entries.forEach((entry) => {
            if (!entry.isIntersecting) {
                return;
            }

            entry.target.classList.add('is-visible');
            currentObserver.unobserve(entry.target);
        });
    }, {
        threshold: 0.18,
        rootMargin: '0px 0px -6% 0px',
    });

    revealTargets.forEach((node) => observer.observe(node));
} else {
    revealTargets.forEach((node) => node.classList.add('is-visible'));
}
