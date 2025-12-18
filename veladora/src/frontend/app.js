// Configuration
// Use relative URL - nginx proxies /api requests to backend
const API_BASE_URL = '/api';
let authToken = localStorage.getItem('authToken');
let currentUsername = localStorage.getItem('username');

// Initialize app
document.addEventListener('DOMContentLoaded', () => {
    setupTabs();
    if (authToken) {
        showDashboard();
        loadProfile();
    } else {
        showAuth();
    }
});

// Tab Management
function setupTabs() {
    // Auth tabs
    document.querySelectorAll('.auth-tabs .tab-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const tab = btn.dataset.tab;
            document.querySelectorAll('.auth-tabs .tab-btn').forEach(b => b.classList.remove('active'));
            document.querySelectorAll('.auth-form').forEach(f => f.classList.remove('active'));
            btn.classList.add('active');
            document.getElementById(`${tab}-form`).classList.add('active');
        });
    });

    // Dashboard tabs
    document.querySelectorAll('nav.tabs .tab-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const tab = btn.dataset.tab;
            document.querySelectorAll('nav.tabs .tab-btn').forEach(b => b.classList.remove('active'));
            document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
            btn.classList.add('active');
            document.getElementById(`${tab}-tab`).classList.add('active');
            
            // Load data when switching tabs
            if (tab === 'orders') loadOrders();
            if (tab === 'bills') {
                loadActiveBill();
                loadBills();
            }
            if (tab === 'payments') {
                loadActiveBill();
            }
            if (tab === 'bartender') loadConversations();
        });
    });
}

// Auth Functions
async function handleLogin() {
    const username = document.getElementById('login-username').value;
    const password = document.getElementById('login-password').value;
    const errorDiv = document.getElementById('auth-error');

    if (!username || !password) {
        errorDiv.textContent = 'Please fill in all fields';
        return;
    }

    try {
        const response = await fetch(`${API_BASE_URL}/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password })
        });

        const data = await response.json();

        if (response.ok) {
            authToken = data.token;
            currentUsername = data.user.username;
            localStorage.setItem('authToken', authToken);
            localStorage.setItem('username', currentUsername);
            showDashboard();
            loadProfile();
            errorDiv.textContent = '';
        } else {
            errorDiv.textContent = data.error || 'Login failed';
        }
    } catch (error) {
        errorDiv.textContent = 'Network error: ' + error.message;
    }
}

async function handleRegister() {
    const username = document.getElementById('register-username').value;
    const password = document.getElementById('register-password').value;
    const errorDiv = document.getElementById('register-error');

    if (!username || !password) {
        errorDiv.textContent = 'Please fill in all fields';
        return;
    }

    try {
        const response = await fetch(`${API_BASE_URL}/register`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password })
        });

        const data = await response.json();

        if (response.ok) {
            authToken = data.token;
            currentUsername = data.user.username;
            localStorage.setItem('authToken', authToken);
            localStorage.setItem('username', currentUsername);
            showDashboard();
            loadProfile();
            errorDiv.textContent = '';
        } else {
            errorDiv.textContent = data.error || 'Registration failed';
        }
    } catch (error) {
        errorDiv.textContent = 'Network error: ' + error.message;
    }
}

function handleLogout() {
    authToken = null;
    currentUsername = null;
    localStorage.removeItem('authToken');
    localStorage.removeItem('username');
    showAuth();
}

function showAuth() {
    document.getElementById('auth-section').classList.remove('hidden');
    document.getElementById('dashboard').classList.add('hidden');
}

function showDashboard() {
    document.getElementById('auth-section').classList.add('hidden');
    document.getElementById('dashboard').classList.remove('hidden');
    document.getElementById('username-display').textContent = currentUsername || 'User';
    // Load active bill if on payments or bills tab
    const activeTab = document.querySelector('nav.tabs .tab-btn.active');
    if (activeTab) {
        const tab = activeTab.dataset.tab;
        if (tab === 'payments' || tab === 'bills') {
            loadActiveBill();
        }
    }
}

// API Helper
async function apiCall(endpoint, method = 'GET', body = null) {
    // Always get fresh token from localStorage to ensure it's up to date
    const token = localStorage.getItem('authToken');
    if (!token) {
        throw new Error('Not authenticated. Please log in.');
    }
    
    const headers = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
    };

    const options = {
        method,
        headers
    };

    if (body) {
        options.body = JSON.stringify(body);
    }

    try {
        const response = await fetch(`${API_BASE_URL}${endpoint}`, options);
        let data;
        
        // Try to parse JSON, but handle cases where response might not be JSON
        const contentType = response.headers.get('content-type');
        if (contentType && contentType.includes('application/json')) {
            data = await response.json();
        } else {
            const text = await response.text();
            throw new Error(text || `HTTP ${response.status}: ${response.statusText}`);
        }
        
        if (!response.ok) {
            throw new Error(data.error || `Request failed: ${response.status} ${response.statusText}`);
        }
        
        return data;
    } catch (error) {
        // Re-throw with more context if it's not already an Error
        if (error instanceof Error) {
            throw error;
        }
        throw new Error(`Network error: ${error.message || error}`);
    }
}

// Profile Functions
async function loadProfile() {
    try {
        const data = await apiCall('/profile');
        document.getElementById('profile-username').textContent = data.username;
        document.getElementById('profile-balance').textContent = data.balance;
        document.getElementById('profile-balance-rubles').textContent = data.balance.toFixed(2);
        document.getElementById('balance-display').textContent = `${data.balance.toFixed(2)} ₽`;
        
        // Display payment links if available
        const paymentLinksDiv = document.getElementById('payment-links');
        if (paymentLinksDiv) {
            paymentLinksDiv.textContent = ''; // Clear existing content
            if (data.payment_links && data.payment_links.length > 0) {
                const heading = document.createElement('h4');
                heading.textContent = 'Your Payment Links:';
                paymentLinksDiv.appendChild(heading);
                
                const list = document.createElement('ul');
                data.payment_links.forEach(link => {
                    const listItem = document.createElement('li');
                    const button = document.createElement('button');
                    button.className = 'link-btn';
                    button.textContent = link;
                    button.setAttribute('data-payment-id', link);
                    button.addEventListener('click', function() {
                        viewBillByPaymentIdInProfile(this.getAttribute('data-payment-id'));
                    });
                    listItem.appendChild(button);
                    list.appendChild(listItem);
                });
                paymentLinksDiv.appendChild(list);
            } else {
                const noLinksMsg = document.createElement('p');
                noLinksMsg.textContent = 'No payment links yet. Pay a bill to generate one.';
                paymentLinksDiv.appendChild(noLinksMsg);
            }
        }
    } catch (error) {
        showMessage('profile-tab', 'Error loading profile: ' + error.message, 'error');
    }
}

function refreshProfile() {
    loadProfile();
}

// Drink Functions
async function orderDrink() {
    const drinkNameInput = document.getElementById('drink-name');
    const messageDiv = document.getElementById('order-message');
    
    if (!drinkNameInput) {
        console.error('Drink name input not found');
        return;
    }
    
    if (!messageDiv) {
        console.error('Message div not found');
        return;
    }

    const drinkName = drinkNameInput.value.trim();

    if (!drinkName) {
        showMessageInDiv(messageDiv, 'Please enter a drink name', 'error');
        return;
    }

    // Show loading state
    showMessageInDiv(messageDiv, 'Placing order...', 'info');

    try {
        const requestBody = { drink_name: drinkName };
        
        const data = await apiCall('/order', 'POST', requestBody);
        showMessageInDiv(messageDiv, `Order placed! Order ID: ${data.order_id}, Bill ID: ${data.bill_id}, Amount: ${data.amount} roubles, Bill Total: ${data.bill_total} roubles`, 'success');
        drinkNameInput.value = '';
        loadProfile(); // Refresh balance
        // Refresh active bill if on bills or payments tab
        if (document.getElementById('bills-tab')?.classList.contains('active') || 
            document.getElementById('payments-tab')?.classList.contains('active')) {
            loadActiveBill();
        }
    } catch (error) {
        console.error('Order error:', error);
        showMessageInDiv(messageDiv, 'Error: ' + (error.message || 'Failed to place order'), 'error');
    }
}

async function loadOrders() {
    const ordersList = document.getElementById('orders-list');
    ordersList.textContent = 'Loading...';

    try {
        const data = await apiCall('/orders');
        ordersList.textContent = ''; // Clear loading message
        if (data.orders && data.orders.length > 0) {
            data.orders.forEach(order => {
                const orderCard = document.createElement('div');
                orderCard.className = 'order-card';
                
                const orderInfo = document.createElement('div');
                orderInfo.className = 'order-card-info';
                
                const orderIdP = document.createElement('p');
                orderIdP.innerHTML = `<strong>Order ID:</strong> ${escapeHtml(String(order.id))}`;
                
                const drinkP = document.createElement('p');
                drinkP.innerHTML = `<strong>Drink:</strong> ${escapeHtml(order.drink_name)}`;
                
                const amountP = document.createElement('p');
                amountP.innerHTML = `<strong>Amount:</strong> ${escapeHtml(String(order.amount))} roubles (${order.amount.toFixed(2)} ₽)`;
                
                const billIdP = document.createElement('p');
                billIdP.innerHTML = `<strong>Bill ID:</strong> ${order.bill_id ? escapeHtml(String(order.bill_id)) : 'N/A'}`;
                
                const statusP = document.createElement('p');
                const statusSpan = document.createElement('span');
                statusSpan.className = `status ${escapeHtml(order.status)}`;
                statusSpan.textContent = order.status;
                statusP.innerHTML = '<strong>Status:</strong> ';
                statusP.appendChild(statusSpan);
                
                const dateP = document.createElement('p');
                dateP.innerHTML = `<strong>Date:</strong> ${escapeHtml(new Date(order.created_at).toLocaleString())}`;
                
                orderInfo.appendChild(orderIdP);
                orderInfo.appendChild(drinkP);
                orderInfo.appendChild(amountP);
                orderInfo.appendChild(billIdP);
                orderInfo.appendChild(statusP);
                orderInfo.appendChild(dateP);
                
                orderCard.appendChild(orderInfo);
                ordersList.appendChild(orderCard);
            });
        } else {
            const noOrdersMsg = document.createElement('p');
            noOrdersMsg.textContent = 'No orders yet.';
            ordersList.appendChild(noOrdersMsg);
        }
    } catch (error) {
        const errorMsg = document.createElement('p');
        errorMsg.className = 'error-message';
        errorMsg.textContent = 'Error loading orders: ' + error.message;
        ordersList.textContent = '';
        ordersList.appendChild(errorMsg);
    }
}


// Bill Functions
async function loadActiveBill(targetDivId = null) {
    // Find the active bill div - prioritize targetDivId if provided, otherwise detect from active tab
    let activeBillDiv = null;
    
    if (targetDivId) {
        // If specific div ID provided, use it
        activeBillDiv = document.getElementById(targetDivId);
    } else {
        // Otherwise, detect from active tab
        if (document.getElementById('payments-tab')?.classList.contains('active')) {
            activeBillDiv = document.getElementById('active-bill-payments');
        } else if (document.getElementById('bills-tab')?.classList.contains('active')) {
            activeBillDiv = document.getElementById('active-bill-bills');
        }
    }
    
    // Fallback: try both divs if still not found
    if (!activeBillDiv) {
        activeBillDiv = document.getElementById('active-bill-payments') || document.getElementById('active-bill-bills');
    }
    
    if (!activeBillDiv) {
        console.error('Active bill div not found. Available divs:', {
            payments: !!document.getElementById('active-bill-payments'),
            bills: !!document.getElementById('active-bill-bills')
        });
        return;
    }
    
    console.log('Loading active bill into div:', activeBillDiv.id);
    activeBillDiv.textContent = 'Loading...';

    try {
        const data = await apiCall('/bill/active');
        console.log('Active bill response:', data); // Debug log
        
        activeBillDiv.textContent = ''; // Clear loading message
        
        if (data && data.bill && data.bill !== null) {
            const bill = data.bill;
            // Ensure orders array exists
            const orders = bill.orders || bill.Orders || [];
            
            console.log('Bill data:', { id: bill.id, amount: bill.amount, ordersCount: orders.length });
            
            const billCard = document.createElement('div');
            billCard.className = 'bill-card active-bill';
            
            const billInfo = document.createElement('div');
            billInfo.className = 'bill-card-info';
            
            const heading = document.createElement('h3');
            heading.textContent = `Active Bill #${bill.id}`;
            billInfo.appendChild(heading);
            
            const amountP = document.createElement('p');
            amountP.innerHTML = `<strong>Total Amount:</strong> ${escapeHtml(String(bill.amount))} roubles`;
            billInfo.appendChild(amountP);
            
            const statusP = document.createElement('p');
            const statusSpan = document.createElement('span');
            statusSpan.className = `status ${escapeHtml(bill.status)}`;
            statusSpan.textContent = bill.status;
            statusP.innerHTML = '<strong>Status:</strong> ';
            statusP.appendChild(statusSpan);
            billInfo.appendChild(statusP);
            
            const commentP = document.createElement('p');
            commentP.innerHTML = `<strong>Comment:</strong> ${bill.comment ? escapeHtml(bill.comment) : '(no comment)'}`;
            billInfo.appendChild(commentP);
            
            const createdP = document.createElement('p');
            createdP.innerHTML = `<strong>Created:</strong> ${escapeHtml(new Date(bill.created_at).toLocaleString())}`;
            billInfo.appendChild(createdP);
            
            billInfo.appendChild(document.createElement('hr'));
            
            const ordersHeading = document.createElement('h4');
            ordersHeading.textContent = 'Orders in this bill:';
            billInfo.appendChild(ordersHeading);
            
            const ordersDiv = document.createElement('div');
            ordersDiv.className = 'orders-in-bill';
            if (orders.length > 0) {
                orders.forEach(order => {
                    const orderItem = document.createElement('div');
                    orderItem.className = 'order-item';
                    
                    const orderText = document.createElement('span');
                    orderText.textContent = `${order.drink_name} - ${order.amount} roubles`;
                    
                    const orderStatus = document.createElement('span');
                    orderStatus.className = `status ${escapeHtml(order.status)}`;
                    orderStatus.textContent = order.status;
                    
                    orderItem.appendChild(orderText);
                    orderItem.appendChild(orderStatus);
                    ordersDiv.appendChild(orderItem);
                });
            } else {
                const noOrdersP = document.createElement('p');
                noOrdersP.textContent = 'No orders yet.';
                ordersDiv.appendChild(noOrdersP);
            }
            billInfo.appendChild(ordersDiv);
            
            const billActions = document.createElement('div');
            billActions.className = 'bill-actions';
            const payButton = document.createElement('button');
            payButton.className = 'close-bill-btn';
            payButton.textContent = 'Pay Bill';
            payButton.addEventListener('click', payBill);
            billActions.appendChild(payButton);
            
            billCard.appendChild(billInfo);
            billCard.appendChild(billActions);
            activeBillDiv.appendChild(billCard);
            
            console.log('Active bill displayed successfully');
        } else {
            console.log('No active bill found in response');
            const noBillMsg = document.createElement('p');
            noBillMsg.textContent = 'No active bill. Order a drink to create one!';
            activeBillDiv.appendChild(noBillMsg);
        }
    } catch (error) {
        console.error('Error loading active bill:', error);
        if (activeBillDiv) {
            activeBillDiv.textContent = '';
            const errorMsg = document.createElement('p');
            errorMsg.className = 'error-message';
            errorMsg.textContent = 'Error loading active bill: ' + error.message;
            activeBillDiv.appendChild(errorMsg);
        }
    }
}

async function payBill() {
    const comment = document.getElementById('pay-bill-comment')?.value || '';
    const messageDiv = document.getElementById('pay-bill-message');

    if (!confirm('Are you sure you want to pay this bill? The money will be deducted from your balance and a payment link will be generated.')) {
        return;
    }

    try {
        const data = await apiCall('/bill/pay', 'POST', { comment });
        showMessageInDiv(messageDiv, `Bill paid successfully! Payment ID: ${data.payment_id}`, 'success');
        if (document.getElementById('pay-bill-comment')) {
            document.getElementById('pay-bill-comment').value = '';
        }
        loadActiveBill();
        loadProfile(); // Refresh balance and payment links
        loadBills(); // Refresh bills list
    } catch (error) {
        showMessageInDiv(messageDiv, 'Error: ' + error.message, 'error');
    }
}

// Helper function to display bill details
function displayBillDetails(bill, result, targetDiv) {
    targetDiv.textContent = ''; // Clear existing content
    
    const orders = bill.orders || bill.Orders || [];
    
    const billCard = document.createElement('div');
    billCard.className = 'bill-card';
    
    const billInfo = document.createElement('div');
    billInfo.className = 'bill-card-info';
    
    const heading = document.createElement('h3');
    heading.textContent = `Bill #${bill.id}`;
    billInfo.appendChild(heading);
    
    const ownerP = document.createElement('p');
    ownerP.innerHTML = `<strong>Owner:</strong> ${escapeHtml(result.username)}`;
    billInfo.appendChild(ownerP);
    
    const amountP = document.createElement('p');
    amountP.innerHTML = `<strong>Amount:</strong> ${escapeHtml(String(bill.amount))} roubles`;
    billInfo.appendChild(amountP);
    
    const statusP = document.createElement('p');
    const statusSpan = document.createElement('span');
    statusSpan.className = `status ${escapeHtml(bill.status)}`;
    statusSpan.textContent = bill.status;
    statusP.innerHTML = '<strong>Status:</strong> ';
    statusP.appendChild(statusSpan);
    billInfo.appendChild(statusP);
    
    const paymentIdP = document.createElement('p');
    paymentIdP.innerHTML = `<strong>Payment ID:</strong> ${escapeHtml(bill.payment_id || '')}`;
    billInfo.appendChild(paymentIdP);
    
    const commentP = document.createElement('p');
    commentP.innerHTML = `<strong>Comment:</strong> ${bill.comment ? escapeHtml(bill.comment) : '(no comment)'}`;
    billInfo.appendChild(commentP);
    
    const createdP = document.createElement('p');
    createdP.innerHTML = `<strong>Created:</strong> ${escapeHtml(new Date(bill.created_at).toLocaleString())}`;
    billInfo.appendChild(createdP);
    
    billInfo.appendChild(document.createElement('hr'));
    
    const ordersHeading = document.createElement('h4');
    ordersHeading.textContent = 'Orders:';
    billInfo.appendChild(ordersHeading);
    
    const ordersDiv = document.createElement('div');
    ordersDiv.className = 'orders-in-bill';
    if (orders.length > 0) {
        orders.forEach(order => {
            const orderItem = document.createElement('div');
            orderItem.className = 'order-item';
            
            const orderText = document.createElement('span');
            orderText.textContent = `${order.drink_name} - ${order.amount} roubles`;
            
            const orderStatus = document.createElement('span');
            orderStatus.className = `status ${escapeHtml(order.status)}`;
            orderStatus.textContent = order.status;
            
            orderItem.appendChild(orderText);
            orderItem.appendChild(orderStatus);
            ordersDiv.appendChild(orderItem);
        });
    } else {
        const noOrdersP = document.createElement('p');
        noOrdersP.textContent = 'No orders.';
        ordersDiv.appendChild(noOrdersP);
    }
    billInfo.appendChild(ordersDiv);
    
    billCard.appendChild(billInfo);
    targetDiv.appendChild(billCard);
}

async function viewBillByPaymentId() {
    const paymentId = document.getElementById('view-bill-payment-id')?.value.trim() || '';
    const messageDiv = document.getElementById('view-bill-message');
    const billViewDiv = document.getElementById('viewed-bill');

    if (!paymentId) {
        showMessageInDiv(messageDiv, 'Please enter a payment ID', 'error');
        return;
    }

    try {
        const result = await apiCall(`/bill/${paymentId}`);
        
        displayBillDetails(result.bill, result, billViewDiv);
        showMessageInDiv(messageDiv, 'Bill loaded successfully', 'success');
    } catch (error) {
        showMessageInDiv(messageDiv, 'Error: ' + error.message, 'error');
        billViewDiv.textContent = '';
    }
}

async function viewBillByPaymentIdInProfile(paymentId) {
    const messageDiv = document.getElementById('view-bill-message');
    const billViewDiv = document.getElementById('viewed-bill');

    if (!paymentId) {
        return;
    }

    const paymentsTab = document.getElementById('payments-tab');
    if (paymentsTab) {
        document.querySelectorAll('nav.tabs .tab-btn').forEach(btn => btn.classList.remove('active'));
        document.querySelectorAll('.tab-content').forEach(content => content.classList.remove('active'));
        document.querySelector(`nav.tabs .tab-btn[data-tab="payments"]`)?.classList.add('active');
        paymentsTab.classList.add('active');
        
        const viewBillSection = document.getElementById('view-bill-section');
        if (viewBillSection) {
            viewBillSection.scrollIntoView({ behavior: 'smooth' });
        }
        
        const paymentIdInput = document.getElementById('view-bill-payment-id');
        if (paymentIdInput) {
            paymentIdInput.value = paymentId;
        }
    }

    try {
        const result = await apiCall(`/bill/${paymentId}`);
        
        displayBillDetails(result.bill, result, billViewDiv);
        showMessageInDiv(messageDiv, 'Bill loaded successfully', 'success');
    } catch (error) {
        showMessageInDiv(messageDiv, 'Error: ' + error.message, 'error');
        billViewDiv.textContent = '';
    }
}

async function loadBills() {
    const billsList = document.getElementById('bills-list');
    billsList.textContent = 'Loading...';

    try {
        const data = await apiCall('/bills');
        billsList.textContent = ''; // Clear loading message
        if (data.bills && data.bills.length > 0) {
            data.bills.forEach(bill => {
                const billCard = document.createElement('div');
                billCard.className = 'bill-card';
                
                const billInfo = document.createElement('div');
                billInfo.className = 'bill-card-info';
                
                const heading = document.createElement('h3');
                heading.textContent = `Bill #${bill.id}`;
                billInfo.appendChild(heading);
                
                const amountP = document.createElement('p');
                amountP.innerHTML = `<strong>Amount:</strong> ${escapeHtml(String(bill.amount))} roubles (${bill.amount.toFixed(2)} ₽)`;
                billInfo.appendChild(amountP);
                
                const statusP = document.createElement('p');
                const statusSpan = document.createElement('span');
                statusSpan.className = `status ${escapeHtml(bill.status)}`;
                statusSpan.textContent = bill.status;
                statusP.innerHTML = '<strong>Status:</strong> ';
                statusP.appendChild(statusSpan);
                billInfo.appendChild(statusP);
                
                const paymentIdP = document.createElement('p');
                paymentIdP.innerHTML = `<strong>Payment ID:</strong> ${bill.payment_id ? escapeHtml(bill.payment_id) : 'Not paid yet'}`;
                billInfo.appendChild(paymentIdP);
                
                const commentP = document.createElement('p');
                commentP.innerHTML = `<strong>Comment:</strong> ${bill.comment ? escapeHtml(bill.comment) : '(no comment)'}`;
                billInfo.appendChild(commentP);
                
                const createdP = document.createElement('p');
                createdP.innerHTML = `<strong>Created:</strong> ${escapeHtml(new Date(bill.created_at).toLocaleString())}`;
                billInfo.appendChild(createdP);
                
                if (bill.payment_id) {
                    const viewBillP = document.createElement('p');
                    viewBillP.innerHTML = '<strong>View Bill:</strong> ';
                    const viewButton = document.createElement('button');
                    viewButton.className = 'link-btn';
                    viewButton.textContent = bill.payment_id;
                    viewButton.setAttribute('data-payment-id', bill.payment_id);
                    viewButton.addEventListener('click', function() {
                        viewBillByPaymentIdInProfile(this.getAttribute('data-payment-id'));
                    });
                    viewBillP.appendChild(viewButton);
                    billInfo.appendChild(viewBillP);
                }
                
                billCard.appendChild(billInfo);
                billsList.appendChild(billCard);
            });
        } else {
            const noBillsMsg = document.createElement('p');
            noBillsMsg.textContent = 'No bills yet.';
            billsList.appendChild(noBillsMsg);
        }
    } catch (error) {
        billsList.textContent = '';
        const errorMsg = document.createElement('p');
        errorMsg.className = 'error-message';
        errorMsg.textContent = 'Error loading bills: ' + error.message;
        billsList.appendChild(errorMsg);
    }
}

// Bartender Functions

async function loadConversations() {
    const chatMessages = document.getElementById('chat-messages');
    chatMessages.textContent = ''; // Clear existing messages

    try {
        const data = await apiCall('/conversations', 'GET');
        if (data.conversations && Array.isArray(data.conversations) && data.conversations.length > 0) {
            data.conversations.forEach(conv => {
                const parts = conv.content.split('\nBartender: ');
                if (parts.length === 2) {
                    addChatMessage(parts[0], 'user');
                    addChatMessage(parts[1], 'bartender');
                } else {
                    addChatMessage(conv.content, 'user');
                }
            });
        } else {
            addChatMessage('No conversations yet. Start chatting with the bartender!', 'system');
        }
    } catch (error) {
        console.error('Error loading conversations:', error);
        addChatMessage('Error loading conversations: ' + error.message, 'error');
    }
}

async function sendMessage() {
    const messageInput = document.getElementById('chat-message');
    const message = messageInput.value.trim();
    const chatMessages = document.getElementById('chat-messages');

    if (!message) return;

    addChatMessage(message, 'user');
    messageInput.value = '';

    try {
        const data = await apiCall('/talk', 'POST', { message });
        addChatMessage(data.message, 'bartender');
    } catch (error) {
        addChatMessage('Error: ' + error.message, 'bartender');
    }
}

function addChatMessage(text, sender) {
    const chatMessages = document.getElementById('chat-messages');
    const messageDiv = document.createElement('div');
    messageDiv.className = `chat-message ${sender}`;
    messageDiv.textContent = text;
    chatMessages.appendChild(messageDiv);
    chatMessages.scrollTop = chatMessages.scrollHeight;
}

async function rememberConversation() {
    const username = document.getElementById('remember-username').value.trim();
    const token = document.getElementById('remember-token').value.trim();
    const messageDiv = document.getElementById('remember-message');
    const conversationsDiv = document.getElementById('remembered-conversations');

    if (!token) {
        showMessageInDiv(messageDiv, 'Please fill in 32-byte token', 'error');
        return;
    }

    if (token.length !== 32) {
        showMessageInDiv(messageDiv, 'Token must be exactly 32 bytes (32 characters)', 'error');
        return;
    }

    try {
        const requestBody = { context_token: token };
        if (username) {
            requestBody.username = username;
        }
        
        const data = await apiCall('/remember', 'POST', requestBody);
        conversationsDiv.textContent = ''; // Clear existing content
        if (data.conversations && data.conversations.length > 0) {
            data.conversations.forEach(conv => {
                const conversationCard = document.createElement('div');
                conversationCard.className = 'conversation-card';
                
                // Parse conversation content
                const parts = conv.content.split('\nBartender: ');
                if (parts.length === 2) {
                    const youP = document.createElement('p');
                    youP.innerHTML = `<strong>You:</strong> ${escapeHtml(parts[0])}`;
                    conversationCard.appendChild(youP);
                    
                    const bartenderP = document.createElement('p');
                    bartenderP.innerHTML = `<strong>Bartender:</strong> ${escapeHtml(parts[1])}`;
                    conversationCard.appendChild(bartenderP);
                } else {
                    const contentP = document.createElement('p');
                    // Replace newlines with <br> but escape HTML first
                    const escapedContent = escapeHtml(conv.content).replace(/\n/g, '<br>');
                    contentP.innerHTML = escapedContent;
                    conversationCard.appendChild(contentP);
                }
                
                const timestampP = document.createElement('p');
                timestampP.className = 'timestamp';
                timestampP.textContent = new Date(conv.created_at).toLocaleString();
                conversationCard.appendChild(timestampP);
                
                conversationsDiv.appendChild(conversationCard);
            });
            showMessageInDiv(messageDiv, `Found ${data.conversations.length} conversation(s)`, 'success');
        } else {
            const noConvMsg = document.createElement('p');
            noConvMsg.textContent = 'No conversations found.';
            conversationsDiv.appendChild(noConvMsg);
            showMessageInDiv(messageDiv, 'No conversations found with that token', 'error');
        }
    } catch (error) {
        showMessageInDiv(messageDiv, 'Error: ' + error.message, 'error');
        conversationsDiv.textContent = '';
    }
}

// Utility Functions
// HTML escaping function to prevent XSS
function escapeHtml(text) {
    if (text == null) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function showMessage(tabId, message, type) {
    const tab = document.getElementById(tabId);
    let messageDiv = tab.querySelector('.message');
    if (!messageDiv) {
        messageDiv = document.createElement('div');
        messageDiv.className = 'message';
        tab.appendChild(messageDiv);
    }
    showMessageInDiv(messageDiv, message, type);
}

function showMessageInDiv(div, message, type) {
    if (!div) {
        console.error('Message div not found');
        return;
    }
    div.textContent = message;
    div.className = `message ${type}`;
    // Don't auto-clear error messages, let user read them
    if (type !== 'error') {
        setTimeout(() => {
            if (div.textContent === message) { // Only clear if message hasn't changed
                div.textContent = '';
                div.className = 'message';
            }
        }, 5000);
    }
}


