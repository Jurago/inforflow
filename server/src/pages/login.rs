pub fn content() -> String {
    r##"
    <div class="login-page">
        <div class="login-card fade-in-up">
            <div class="login-brand">
                <h2>Inforflow</h2>
                <p>Faça login para ver os dados do ISP</p>
            </div>
            <p class="login-banner" id="login-banner" hidden>
                A coleta continua ativa. Sem sessão autenticada a interface não exibe métricas.
            </p>
            <label class="login-label" for="login-user">Usuário</label>
            <input type="text" id="login-user" class="login-input" placeholder="admin" autocomplete="username" autofocus>
            <label class="login-label" for="login-pass">Senha</label>
            <input type="password" id="login-pass" class="login-input" placeholder="••••••••" autocomplete="current-password">
            <button type="button" id="login-btn" class="login-btn">Entrar</button>
            <p class="login-hint login-error" id="login-error"></p>
            <p class="login-hint">Os dados NetFlow/SNMP/BGP não são afetados pelo login.</p>
        </div>
    </div>
    <script src="/static/js/auth.js?v=20260825a"></script>
    "##.to_string()
}

pub fn render_standalone() -> String {
    format!(
        r##"<!DOCTYPE html>
<html lang="pt-BR">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Login — Inforflow</title>
    <link rel="stylesheet" href="/static/css/inforflow.css">
</head>
<body><div class="app-shell app-shell-login">{content}</div></body>
</html>"##,
        content = content()
    )
}
