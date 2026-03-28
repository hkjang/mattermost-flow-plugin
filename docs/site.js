const revealTargets = Array.from(document.querySelectorAll('.reveal'));
const yearNode = document.getElementById('year');

if (yearNode) {
    yearNode.textContent = String(new Date().getFullYear());
}

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
