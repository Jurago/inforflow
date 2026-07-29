(function () {
    const userInput = document.getElementById('login-user');
    const passInput = document.getElementById('login-pass');
    const btn = document.getElementById('login-btn');
    const err = document.getElementById('login-error');

    async function checkAlreadyLoggedIn() {
        const token = localStorage.getItem('inforflow_api_token');
        if (!token) return;
        try {
            const res = await fetch('/api/auth/check', { headers: { 'X-API-Token': token } });
            const data = await res.json();
            if (data.ok) window.location.href = '/';
        } catch (e) { /* ignore */ }
    }

    async function doLogin() {
        const username = (userInput && userInput.value || '').trim();
        const password = passInput ? passInput.value : '';
        if (!username || !password) {
            if (err) err.textContent = 'Informe usuário e senha';
            return;
        }
        if (err) err.textContent = '';
        if (btn) { btn.disabled = true; btn.textContent = 'Entrando…'; }
        try {
            const res = await fetch('/api/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, password })
            });
            const data = await res.json();
            if (data.ok && data.token) {
                localStorage.setItem('inforflow_api_token', data.token);
                window.location.href = '/';
                return;
            }
            if (err) err.textContent = data.error || 'Usuário ou senha inválidos';
        } catch (e) {
            if (err) err.textContent = 'Erro ao conectar ao servidor';
        } finally {
            if (btn) { btn.disabled = false; btn.textContent = 'Entrar'; }
        }
    }

    btn && btn.addEventListener('click', doLogin);
    passInput && passInput.addEventListener('keydown', e => { if (e.key === 'Enter') doLogin(); });
    userInput && userInput.addEventListener('keydown', e => { if (e.key === 'Enter') passInput && passInput.focus(); });

    checkAlreadyLoggedIn();
})();
