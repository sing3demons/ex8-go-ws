// Chat Application JavaScript
class ChatApp {
    constructor() {
        this.ws = null;
        this.isConnected = false;
        this.currentUser = null;
        this.currentRoom = 'general';
        this.rooms = new Set(['general']);
        this.users = new Set();
        this.messageHistory = [];
        
        this.initializeElements();
        this.bindEvents();
        this.showLoginScreen();
    }

    initializeElements() {
        // Login elements
        this.loginScreen = document.getElementById('loginScreen');
        this.chatInterface = document.getElementById('chatInterface');
        this.usernameInput = document.getElementById('usernameInput');
        this.joinBtn = document.getElementById('joinBtn');

        // Chat elements
        this.messagesContainer = document.getElementById('messagesContainer');
        this.messages = document.getElementById('messages');
        this.messageInput = document.getElementById('messageInput');
        this.sendBtn = document.getElementById('sendBtn');

        // Status elements
        this.connectionStatus = document.getElementById('connectionStatus');
        this.statusText = document.getElementById('statusText');
        this.currentUsername = document.getElementById('currentUsername');
        this.userInfo = document.getElementById('userInfo');

        // Room elements
        this.roomsList = document.getElementById('roomsList');
        this.currentRoomName = document.getElementById('currentRoomName');
        this.roomUserCount = document.getElementById('roomUserCount');
        this.createRoomBtn = document.getElementById('createRoomBtn');

        // User elements
        this.usersList = document.getElementById('usersList');

        // Modal elements
        this.createRoomModal = document.getElementById('createRoomModal');
        this.historyModal = document.getElementById('historyModal');
        this.searchModal = document.getElementById('searchModal');

        // Notification container
        this.notifications = document.getElementById('notifications');
    }

    bindEvents() {
        // Login events
        this.joinBtn.addEventListener('click', () => this.joinChat());
        this.usernameInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this.joinChat();
        });

        // Message events
        this.sendBtn.addEventListener('click', () => this.sendMessage());
        this.messageInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this.sendMessage();
        });

        // Room events
        this.createRoomBtn.addEventListener('click', () => this.showCreateRoomModal());

        // Modal events
        this.bindModalEvents();

        // History and search events
        document.getElementById('historyBtn').addEventListener('click', () => this.showHistoryModal());
        document.getElementById('searchBtn').addEventListener('click', () => this.showSearchModal());
    }

    bindModalEvents() {
        // Create Room Modal
        const createRoomModal = document.getElementById('createRoomModal');
        const closeCreateRoomModal = document.getElementById('closeCreateRoomModal');
        const cancelCreateRoom = document.getElementById('cancelCreateRoom');
        const confirmCreateRoom = document.getElementById('confirmCreateRoom');
        const roomNameInput = document.getElementById('roomNameInput');

        closeCreateRoomModal.addEventListener('click', () => this.hideModal(createRoomModal));
        cancelCreateRoom.addEventListener('click', () => this.hideModal(createRoomModal));
        confirmCreateRoom.addEventListener('click', () => this.createRoom());
        
        roomNameInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this.createRoom();
        });

        // History Modal
        const historyModal = document.getElementById('historyModal');
        const closeHistoryModal = document.getElementById('closeHistoryModal');
        const loadMoreHistory = document.getElementById('loadMoreHistory');

        closeHistoryModal.addEventListener('click', () => this.hideModal(historyModal));
        loadMoreHistory.addEventListener('click', () => this.loadMessageHistory());

        // Search Modal
        const searchModal = document.getElementById('searchModal');
        const closeSearchModal = document.getElementById('closeSearchModal');
        const performSearch = document.getElementById('performSearch');
        const searchInput = document.getElementById('searchInput');

        closeSearchModal.addEventListener('click', () => this.hideModal(searchModal));
        performSearch.addEventListener('click', () => this.searchMessages());
        
        searchInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this.searchMessages();
        });

        // Close modals when clicking outside
        [createRoomModal, historyModal, searchModal].forEach(modal => {
            modal.addEventListener('click', (e) => {
                if (e.target === modal) this.hideModal(modal);
            });
        });
    }

    showLoginScreen() {
        this.loginScreen.style.display = 'flex';
        this.chatInterface.style.display = 'none';
        this.usernameInput.focus();
    }

    showChatInterface() {
        this.loginScreen.style.display = 'none';
        this.chatInterface.style.display = 'flex';
        this.userInfo.style.display = 'block';
        this.messageInput.focus();
    }

    joinChat() {
        const username = this.usernameInput.value.trim();
        
        if (!this.validateUsername(username)) {
            this.showNotification('Please enter a valid username (3-50 characters, letters, numbers, underscore, dash only)', 'error');
            return;
        }

        this.currentUser = username;
        this.currentUsername.textContent = username;
        this.connectWebSocket();
    }

    validateUsername(username) {
        if (!username || username.length < 3 || username.length > 50) {
            return false;
        }
        
        // Allow letters, numbers, underscore, and dash
        const validPattern = /^[a-zA-Z0-9_-]+$/;
        return validPattern.test(username);
    }

    connectWebSocket() {
        const wsUrl = `ws://${window.location.host}/ws`;
        
        try {
            this.ws = new WebSocket(wsUrl);
            
            this.ws.onopen = () => {
                this.onWebSocketOpen();
            };
            
            this.ws.onmessage = (event) => {
                this.onWebSocketMessage(event);
            };
            
            this.ws.onclose = (event) => {
                this.onWebSocketClose(event);
            };
            
            this.ws.onerror = (error) => {
                this.onWebSocketError(error);
            };
            
        } catch (error) {
            this.showNotification('Failed to connect to chat server', 'error');
            console.error('WebSocket connection error:', error);
        }
    }

    onWebSocketOpen() {
        this.isConnected = true;
        this.updateConnectionStatus(true);
        this.showChatInterface();
        
        // Send username to server
        this.sendToServer({
            type: 'join',
            username: this.currentUser
        });
        
        this.showNotification('Connected to chat server', 'success');
    }

    onWebSocketMessage(event) {
        try {
            const data = JSON.parse(event.data);
            this.handleServerMessage(data);
        } catch (error) {
            // Handle plain text messages (for backward compatibility)
            this.displayMessage({
                type: 'message',
                content: event.data,
                sender: 'System',
                timestamp: new Date().toISOString()
            });
        }
    }

    onWebSocketClose(event) {
        this.isConnected = false;
        this.updateConnectionStatus(false);
        
        if (event.code !== 1000) { // Not a normal closure
            this.showNotification('Connection lost. Attempting to reconnect...', 'warning');
            setTimeout(() => this.connectWebSocket(), 3000);
        }
    }

    onWebSocketError(error) {
        console.error('WebSocket error:', error);
        this.showNotification('Connection error occurred', 'error');
    }

    handleServerMessage(data) {
        switch (data.type) {
            case 'message':
                this.displayMessage(data);
                break;
            case 'user_joined':
                this.handleUserJoined(data);
                break;
            case 'user_left':
                this.handleUserLeft(data);
                break;
            case 'room_created':
                this.handleRoomCreated(data);
                break;
            case 'room_joined':
                this.handleRoomJoined(data);
                break;
            case 'room_left':
                this.handleRoomLeft(data);
                break;
            case 'users_list':
                this.updateUsersList(data.users);
                break;
            case 'rooms_list':
                this.updateRoomsList(data.rooms);
                break;
            case 'history':
                this.displayHistory(data.messages);
                break;
            case 'search_results':
                this.displaySearchResults(data.messages);
                break;
            case 'error':
                this.showNotification(data.message, 'error');
                break;
            case 'system':
                this.displaySystemMessage(data.message);
                break;
            default:
                console.log('Unknown message type:', data);
        }
    }

    sendMessage() {
        const content = this.messageInput.value.trim();
        
        if (!content || !this.isConnected) {
            return;
        }

        if (content.length > 500) {
            this.showNotification('Message too long (max 500 characters)', 'error');
            return;
        }

        // Check if it's a command
        if (content.startsWith('/')) {
            this.handleCommand(content);
        } else {
            this.sendToServer({
                type: 'message',
                content: content,
                room: this.currentRoom
            });
        }

        this.messageInput.value = '';
    }

    handleCommand(command) {
        const parts = command.split(' ');
        const cmd = parts[0].toLowerCase();
        const args = parts.slice(1);

        switch (cmd) {
            case '/join':
                if (args.length > 0) {
                    this.joinRoom(args[0]);
                } else {
                    this.showNotification('Usage: /join <room_name>', 'error');
                }
                break;
            case '/create':
                if (args.length > 0) {
                    this.createRoomCommand(args[0]);
                } else {
                    this.showNotification('Usage: /create <room_name>', 'error');
                }
                break;
            case '/leave':
                this.leaveRoom();
                break;
            case '/history':
                const limit = args.length > 0 ? parseInt(args[0]) : 25;
                this.requestHistory(limit);
                break;
            case '/search':
                if (args.length > 0) {
                    this.searchMessagesCommand(args.join(' '));
                } else {
                    this.showNotification('Usage: /search <query>', 'error');
                }
                break;
            case '/myhistory':
                const myLimit = args.length > 0 ? parseInt(args[0]) : 25;
                this.requestMyHistory(myLimit);
                break;
            default:
                // Send command to server
                this.sendToServer({
                    type: 'command',
                    command: command,
                    room: this.currentRoom
                });
        }
    }

    sendToServer(data) {
        if (this.isConnected && this.ws) {
            this.ws.send(JSON.stringify(data));
        }
    }

    displayMessage(data) {
        const messageDiv = document.createElement('div');
        messageDiv.className = 'message';
        
        // Determine message type
        if (data.sender === this.currentUser) {
            messageDiv.classList.add('own');
        } else if (data.type === 'system') {
            messageDiv.classList.add('system');
        } else {
            messageDiv.classList.add('other');
        }

        const timestamp = new Date(data.timestamp).toLocaleTimeString();
        
        let messageHTML = '';
        if (data.type !== 'system' && data.sender !== this.currentUser) {
            messageHTML += `<div class="message-header">${data.sender}</div>`;
        }
        
        messageHTML += `<div class="message-content">${this.escapeHtml(data.content)}</div>`;
        messageHTML += `<div class="message-time">${timestamp}</div>`;
        
        messageDiv.innerHTML = messageHTML;
        
        this.messages.appendChild(messageDiv);
        this.scrollToBottom();
        
        // Store in history
        this.messageHistory.push(data);
        if (this.messageHistory.length > 1000) {
            this.messageHistory = this.messageHistory.slice(-1000);
        }
    }

    displaySystemMessage(message) {
        this.displayMessage({
            type: 'system',
            content: message,
            sender: 'System',
            timestamp: new Date().toISOString()
        });
    }

    handleUserJoined(data) {
        this.users.add(data.username);
        this.updateUsersList(Array.from(this.users));
        this.displaySystemMessage(`${data.username} joined the room`);
    }

    handleUserLeft(data) {
        this.users.delete(data.username);
        this.updateUsersList(Array.from(this.users));
        this.displaySystemMessage(`${data.username} left the room`);
    }

    handleRoomCreated(data) {
        this.rooms.add(data.room);
        this.updateRoomsList(Array.from(this.rooms));
        this.showNotification(`Room "${data.room}" created successfully`, 'success');
    }

    handleRoomJoined(data) {
        this.currentRoom = data.room;
        this.currentRoomName.textContent = data.room;
        this.clearMessages();
        this.updateRoomsList(Array.from(this.rooms));
        this.displaySystemMessage(`Joined room: ${data.room}`);
    }

    handleRoomLeft(data) {
        this.displaySystemMessage(`Left room: ${data.room}`);
    }

    updateUsersList(users) {
        this.users = new Set(users);
        this.usersList.innerHTML = '';
        
        users.forEach(user => {
            const userDiv = document.createElement('div');
            userDiv.className = 'user-item';
            userDiv.innerHTML = `
                <div class="status-dot"></div>
                <span>${this.escapeHtml(user)}</span>
            `;
            this.usersList.appendChild(userDiv);
        });
        
        this.roomUserCount.textContent = `${users.length} user${users.length !== 1 ? 's' : ''}`;
    }

    updateRoomsList(rooms) {
        this.rooms = new Set(rooms);
        this.roomsList.innerHTML = '';
        
        rooms.forEach(room => {
            const roomDiv = document.createElement('div');
            roomDiv.className = 'room-item';
            if (room === this.currentRoom) {
                roomDiv.classList.add('active');
            }
            
            roomDiv.textContent = room;
            roomDiv.addEventListener('click', () => this.joinRoom(room));
            
            this.roomsList.appendChild(roomDiv);
        });
    }

    joinRoom(roomName) {
        if (roomName === this.currentRoom) {
            return;
        }
        
        this.sendToServer({
            type: 'join_room',
            room: roomName
        });
    }

    leaveRoom() {
        if (this.currentRoom === 'general') {
            this.showNotification('Cannot leave the general room', 'error');
            return;
        }
        
        this.sendToServer({
            type: 'leave_room',
            room: this.currentRoom
        });
        
        this.currentRoom = 'general';
        this.currentRoomName.textContent = 'general';
    }

    createRoomCommand(roomName) {
        if (!this.validateRoomName(roomName)) {
            this.showNotification('Invalid room name (3-50 characters, letters, numbers, underscore, dash only)', 'error');
            return;
        }
        
        this.sendToServer({
            type: 'create_room',
            room: roomName
        });
    }

    validateRoomName(roomName) {
        if (!roomName || roomName.length < 3 || roomName.length > 50) {
            return false;
        }
        
        const validPattern = /^[a-zA-Z0-9_-]+$/;
        return validPattern.test(roomName);
    }

    // Modal functions
    showCreateRoomModal() {
        this.showModal(this.createRoomModal);
        document.getElementById('roomNameInput').focus();
    }

    createRoom() {
        const roomName = document.getElementById('roomNameInput').value.trim();
        
        if (!this.validateRoomName(roomName)) {
            this.showNotification('Invalid room name (3-50 characters, letters, numbers, underscore, dash only)', 'error');
            return;
        }
        
        this.createRoomCommand(roomName);
        this.hideModal(this.createRoomModal);
        document.getElementById('roomNameInput').value = '';
    }

    showHistoryModal() {
        this.showModal(this.historyModal);
        this.loadMessageHistory();
    }

    loadMessageHistory() {
        const limit = parseInt(document.getElementById('historyLimit').value) || 25;
        this.requestHistory(limit);
    }

    requestHistory(limit = 25) {
        this.sendToServer({
            type: 'get_history',
            room: this.currentRoom,
            limit: limit
        });
    }

    requestMyHistory(limit = 25) {
        this.sendToServer({
            type: 'get_my_history',
            limit: limit
        });
    }

    displayHistory(messages) {
        const historyMessages = document.getElementById('historyMessages');
        historyMessages.innerHTML = '';
        
        messages.forEach(msg => {
            const messageDiv = document.createElement('div');
            messageDiv.className = 'history-message';
            
            const timestamp = new Date(msg.timestamp).toLocaleString();
            messageDiv.innerHTML = `
                <div class="message-meta">
                    <strong>${this.escapeHtml(msg.sender)}</strong> 
                    in <em>${this.escapeHtml(msg.room || 'general')}</em> 
                    at ${timestamp}
                </div>
                <div class="message-content">${this.escapeHtml(msg.content)}</div>
            `;
            
            historyMessages.appendChild(messageDiv);
        });
    }

    showSearchModal() {
        this.showModal(this.searchModal);
        document.getElementById('searchInput').focus();
    }

    searchMessages() {
        const query = document.getElementById('searchInput').value.trim();
        
        if (!query) {
            this.showNotification('Please enter a search query', 'error');
            return;
        }
        
        this.searchMessagesCommand(query);
    }

    searchMessagesCommand(query) {
        this.sendToServer({
            type: 'search_messages',
            query: query,
            room: this.currentRoom
        });
    }

    displaySearchResults(messages) {
        const searchResults = document.getElementById('searchResults');
        searchResults.innerHTML = '';
        
        if (messages.length === 0) {
            searchResults.innerHTML = '<div class="text-center text-muted">No messages found</div>';
            return;
        }
        
        messages.forEach(msg => {
            const messageDiv = document.createElement('div');
            messageDiv.className = 'search-result';
            
            const timestamp = new Date(msg.timestamp).toLocaleString();
            messageDiv.innerHTML = `
                <div class="message-meta">
                    <strong>${this.escapeHtml(msg.sender)}</strong> 
                    in <em>${this.escapeHtml(msg.room || 'general')}</em> 
                    at ${timestamp}
                </div>
                <div class="message-content">${this.escapeHtml(msg.content)}</div>
            `;
            
            searchResults.appendChild(messageDiv);
        });
    }

    showModal(modal) {
        modal.classList.add('show');
    }

    hideModal(modal) {
        modal.classList.remove('show');
    }

    updateConnectionStatus(connected) {
        this.isConnected = connected;
        
        if (connected) {
            this.connectionStatus.classList.remove('disconnected');
            this.connectionStatus.classList.add('connected');
            this.statusText.textContent = 'Connected';
        } else {
            this.connectionStatus.classList.remove('connected');
            this.connectionStatus.classList.add('disconnected');
            this.statusText.textContent = 'Disconnected';
        }
    }

    clearMessages() {
        this.messages.innerHTML = '';
    }

    scrollToBottom() {
        this.messagesContainer.scrollTop = this.messagesContainer.scrollHeight;
    }

    showNotification(message, type = 'info') {
        const notification = document.createElement('div');
        notification.className = `notification ${type}`;
        notification.textContent = message;
        
        this.notifications.appendChild(notification);
        
        // Auto remove after 5 seconds
        setTimeout(() => {
            if (notification.parentNode) {
                notification.parentNode.removeChild(notification);
            }
        }, 5000);
        
        // Allow manual close
        notification.addEventListener('click', () => {
            if (notification.parentNode) {
                notification.parentNode.removeChild(notification);
            }
        });
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// Initialize the chat application when the page loads
document.addEventListener('DOMContentLoaded', () => {
    window.chatApp = new ChatApp();
});