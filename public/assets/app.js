let currentAuthMode = 'login';
let ws = null;
let currentUsername = "";
let wsToken = "";

function showToast(msg) {
    const toast = document.getElementById('toast');
    toast.textContent = msg;
    toast.classList.add('show');
    setTimeout(() => { toast.classList.remove('show'); }, 3000);
}

async function initSessionData() {
    try {
        const res = await fetch('/app/chat/api/session');
        if (res.ok) {
            const data = await res.json();
            currentUsername = data.username;
            wsToken = data.token;
            if (wsToken) sessionStorage.setItem("wsToken", wsToken);
        } else {
            sessionStorage.removeItem("wsToken");
        }
    } catch (e) { }
}

async function showView(viewId, modeCtx = null, pushState = true) {
    document.querySelectorAll('.view').forEach(v => v.classList.remove('active'));
    document.getElementById(viewId).classList.add('active');

    if (viewId === 'welcome-view') {
        const targetPath = window.location.pathname.startsWith('/app/chat') ? '/app/chat/' : '/';
        if (pushState && window.location.pathname !== targetPath) history.pushState({}, '', targetPath);
        renderWelcomeActions();
    } else if (viewId === 'auth-view') {
        if (modeCtx === 'register') {
            setAuthMode('register');
        } else {
            setAuthMode('login');
        }
    } else if (viewId === 'chat-view') {
        const targetPath = window.location.pathname.startsWith('/app/chat') ? '/app/chat/room' : '/room';
        if (pushState && window.location.pathname !== targetPath) history.pushState({}, '', targetPath);
        if (!ws || (ws.readyState !== WebSocket.OPEN && ws.readyState !== WebSocket.CONNECTING)) {
            connectWebSocket();
        }
    }
}

function renderWelcomeActions() {
    const actionsContainer = document.getElementById('welcome-actions');
    if (currentUsername) {
        actionsContainer.innerHTML = `
            <button class="btn-primary" onclick="showView('chat-view')">Ir al chat</button>
        `;
    } else {
        actionsContainer.innerHTML = `
            <button class="btn-primary" onclick="showView('auth-view', 'login')">Iniciar Sesión</button>
            <button class="btn-ghost" onclick="showView('auth-view', 'register')">Registrarse</button>
        `;
    }
}

function toggleAuthMode() {
    setAuthMode(currentAuthMode === 'login' ? 'register' : 'login');
}

function setAuthMode(mode) {
    currentAuthMode = mode;
    const isReg = mode === 'register';

    document.getElementById('auth-title').innerText = isReg ? 'Crear Cuenta' : 'Iniciar Sesión';
    document.getElementById('auth-submit-btn').innerText = isReg ? 'Registrarse' : 'Entrar';
    document.getElementById('group-email').style.display = isReg ? 'block' : 'none';
    document.getElementById('auth-email').required = isReg;

    document.getElementById('auth-toggle-text').innerHTML = isReg
        ? `¿Ya tienes cuenta? <a onclick="toggleAuthMode()">Inicia Sesión</a>`
        : `¿No tienes cuenta? <a onclick="toggleAuthMode()">Regístrate</a>`;
}

async function handleAuth(e) {
    e.preventDefault();
    const username = document.getElementById('auth-username').value;
    const password = document.getElementById('auth-password').value;
    const email = document.getElementById('auth-email').value;

    const payload = { username, password };
    if (currentAuthMode === 'register') payload.email = email;

    const route = currentAuthMode === 'register' ? '/app/chat/api/register' : '/app/chat/api/login';

    try {
        const response = await fetch(route, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });

        if (response.ok) {
            if (currentAuthMode === 'register') {
                showToast("Registro exitoso. Iniciando sesión...");
                setAuthMode('login');
                await handleAuth(new Event('submit'));
            } else {
                showToast("Sesión iniciada");
                const loginData = await response.json();
                if (loginData.token) {
                    wsToken = loginData.token;
                    sessionStorage.setItem("wsToken", wsToken);
                }
                window.location.href = window.location.pathname.startsWith('/app/chat') ? '/app/chat/room' : '/room';
            }
        } else {
            showToast("Error de autenticación. Revisa tus datos.");
        }
    } catch (err) {
        console.error("Auth fetch failed:", err);
        showToast("Error de red.");
    }
}

async function logout() {
    try {
        await fetch('/app/chat/api/logout', { method: 'POST' });
    } catch (e) {
        console.error("Error during logout:", e);
    }

    if (ws) {
        ws.close();
        ws = null;
    }
    wsToken = "";
    sessionStorage.removeItem("wsToken");
    showToast("Sesión cerrada.");
    window.location.href = window.location.pathname.startsWith('/app/chat') ? '/app/chat/' : '/';
}

function connectWebSocket() {
    const loc = window.location;
    let wsUri = "ws:";
    if (loc.protocol === "https:") {
        wsUri = "wss:";
    }
    wsUri += "//" + loc.host + "/app/chat/ws";

    if (!wsToken) {
        wsToken = sessionStorage.getItem("wsToken") || "";
    }

    if (wsToken) {
        wsUri += "?token=" + encodeURIComponent(wsToken);
    }

    ws = new WebSocket(wsUri);

    let pingInterval;

    ws.onopen = () => {
        showToast("Conectado al chat");
        document.getElementById('messages-container').innerHTML = '';

        pingInterval = setInterval(() => {
            if (ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({ type: "ping" }));
            }
        }, 15000);
    };

    ws.onmessage = (evt) => {
        try {
            const data = JSON.parse(evt.data);

            if (data.type === "users") {
                const usersList = document.getElementById('active-users-list');
                const usersCount = document.getElementById('users-count');

                usersList.innerHTML = '';
                data.users.forEach(user => {
                    const li = document.createElement('li');
                    li.textContent = user;
                    if (user === currentUsername) {
                        li.style.fontWeight = 'bold';
                        li.textContent += ' (Tú)';
                    }
                    usersList.appendChild(li);
                });

                usersCount.textContent = data.users.length;

            } else if (data.type === "chat") {
                const messagesContainer = document.getElementById('messages-container');
                const author = data.username || "Sistema";
                const text = data.content || "";

                const msgDiv = document.createElement('div');
                msgDiv.className = `message ${author === currentUsername ? 'message-self' : 'message-other'}`;

                const authorSpan = document.createElement('span');
                authorSpan.className = 'message-author';
                authorSpan.textContent = author;

                const textSpan = document.createElement('span');
                textSpan.textContent = text;

                msgDiv.appendChild(authorSpan);
                msgDiv.appendChild(textSpan);
                messagesContainer.appendChild(msgDiv);

                messagesContainer.scrollTop = messagesContainer.scrollHeight;
            }
        } catch (e) {
            console.error("Error parsing WS message:", e, evt.data);
        }
    };

    ws.onclose = () => {
        if (pingInterval) clearInterval(pingInterval);
        showToast("Desconectado del servidor de chat.");
    };

    ws.onerror = (e) => {
        if (pingInterval) clearInterval(pingInterval);
        console.error("WebSocket Error:", e);
    };
}

function sendMessage(e) {
    e.preventDefault();
    if (!ws || ws.readyState !== WebSocket.OPEN) {
        showToast("WS no conectado.");
        return;
    }
    const input = document.getElementById('message-input');
    if (input.value.trim() !== '') {
        ws.send(input.value);
        input.value = '';
    }
}

window.addEventListener('popstate', () => {
    routeBasedOnURL(false);
});

async function routeBasedOnURL(pushState = true) {
    const path = window.location.pathname;

    if (path.endsWith('/room')) {
        await showView('chat-view', null, pushState);
    } else {
        await showView('welcome-view', null, pushState);
    }
}

// Initial check 
window.onload = async () => {
    await initSessionData();
    routeBasedOnURL(false);
};
