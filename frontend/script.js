let currentToken = null;
let currentUser = null;

// API Configuration
const API_BASE = 'http://localhost:8080/api';

// Login function
document.getElementById('loginForm').addEventListener('submit', async function(e) {
    e.preventDefault();
    
    const username = document.getElementById('username').value;
    const role = document.getElementById('role').value;
    
    try {
        const response = await fetch(`${API_BASE}/auth/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, role })
        });
        
        if (!response.ok) {
            throw new Error('Login failed');
        }
        
        const data = await response.json();
        currentToken = data.token;
        currentUser = data.user;
        
        // Update UI
        document.getElementById('current-user').textContent = 
            `${currentUser.username} (${currentUser.role})`;
        
        // Update permissions
        updatePermissions(currentUser.role);
        
        // Load items
        loadItems();
        
        // Show success message
        alert('Login successful!');
        
        // Close dropdown
        bootstrap.Dropdown.getInstance(document.getElementById('loginDropdown')).hide();
        
    } catch (error) {
        alert('Login failed: ' + error.message);
    }
});

// Update permissions display
function updatePermissions(role) {
    const permissions = {
        admin: ['View items', 'Add items', 'Edit items', 'Delete items', 'View history', 'Export data'],
        manager: ['View items', 'Add items', 'Edit items', 'View history'],
        viewer: ['View items'],
        auditor: ['View items', 'View history']
    };
    
    const list = document.getElementById('permissions-list');
    list.innerHTML = '';
    
    permissions[role].forEach(perm => {
        const li = document.createElement('li');
        li.innerHTML = `<i class="bi bi-check-circle text-success"></i> ${perm}`;
        list.appendChild(li);
    });
    
    // Show/hide buttons based on role
    const addButton = document.getElementById('add-item-btn');
    if (role === 'viewer') {
        addButton.style.display = 'none';
    } else {
        addButton.style.display = 'block';
    }
}

// Navigation
function showSection(sectionId) {
    // Hide all sections
    document.querySelectorAll('.section').forEach(section => {
        section.classList.add('d-none');
    });
    
    // Remove active class from all nav items
    document.querySelectorAll('.list-group-item').forEach(item => {
        item.classList.remove('active');
    });
    
    // Show selected section
    document.getElementById(`${sectionId}-section`).classList.remove('d-none');
    
    // Set active nav item
    event.target.classList.add('active');
    
    // Load data if needed
    if (sectionId === 'items') {
        loadItems();
    }
}

// Load items
async function loadItems() {
    if (!currentToken) {
        alert('Please login first');
        return;
    }
    
    try {
        const response = await fetch(`${API_BASE}/items`, {
            headers: {
                'Authorization': `Bearer ${currentToken}`
            }
        });
        
        if (!response.ok) {
            if (response.status === 403) {
                throw new Error('You do not have permission to view items');
            }
            throw new Error('Failed to load items');
        }
        
        const items = await response.json();
        renderItems(items);
        
    } catch (error) {
        alert(error.message);
    }
}

// Render items table
function renderItems(items) {
    const tbody = document.getElementById('items-table-body');
    tbody.innerHTML = '';
    
    if (items.length === 0) {
        tbody.innerHTML = '<tr><td colspan="8" class="text-center">No items found</td></tr>';
        return;
    }
    
    items.forEach(item => {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td>${item.id}</td>
            <td>${item.name}</td>
            <td>
                <span class="badge ${item.quantity > 10 ? 'bg-success' : 'bg-warning'}">
                    ${item.quantity}
                </span>
            </td>
            <td>$${item.price.toFixed(2)}</td>
            <td>${item.location || 'N/A'}</td>
            <td>${item.created_by}</td>
            <td>${new Date(item.updated_at).toLocaleString()}</td>
            <td>
                <button class="btn btn-sm btn-outline-primary me-1" onclick="viewItemHistory(${item.id})" title="View History">
                    <i class="bi bi-clock-history"></i>
                </button>
                <button class="btn btn-sm btn-outline-warning me-1" onclick="showEditItemModal(${item.id})" 
                    ${currentUser.role === 'viewer' ? 'disabled' : ''} title="Edit">
                    <i class="bi bi-pencil"></i>
                </button>
                <button class="btn btn-sm btn-outline-danger" onclick="deleteItem(${item.id})" 
                    ${currentUser.role !== 'admin' ? 'disabled' : ''} title="Delete">
                    <i class="bi bi-trash"></i>
                </button>
            </td>
        `;
        tbody.appendChild(row);
    });
}

// Add item
async function addItem() {
    if (!currentToken) {
        alert('Please login first');
        return;
    }
    
    const form = document.getElementById('add-item-form');
    const formData = new FormData(form);
    const data = Object.fromEntries(formData.entries());
    
    // Convert types
    data.quantity = parseInt(data.quantity);
    data.price = parseFloat(data.price);
    
    try {
        const response = await fetch(`${API_BASE}/items`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${currentToken}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(data)
        });
        
        if (!response.ok) {
            throw new Error('Failed to add item');
        }
        
        // Close modal and reload items
        bootstrap.Modal.getInstance(document.getElementById('addItemModal')).hide();
        loadItems();
        alert('Item added successfully!');
        
    } catch (error) {
        alert(error.message);
    }
}

// Edit item
async function updateItem() {
    if (!currentToken) {
        alert('Please login first');
        return;
    }
    
    const itemId = document.getElementById('edit-item-id').value;
    const form = document.getElementById('edit-item-form');
    const formData = new FormData(form);
    const data = {};
    
    formData.forEach((value, key) => {
        if (value) {
            if (key === 'quantity') data[key] = parseInt(value);
            else if (key === 'price') data[key] = parseFloat(value);
            else data[key] = value;
        }
    });
    
    try {
        const response = await fetch(`${API_BASE}/items/${itemId}`, {
            method: 'PUT',
            headers: {
                'Authorization': `Bearer ${currentToken}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(data)
        });
        
        if (!response.ok) {
            throw new Error('Failed to update item');
        }
        
        // Close modal and reload items
        bootstrap.Modal.getInstance(document.getElementById('editItemModal')).hide();
        loadItems();
        alert('Item updated successfully!');
        
    } catch (error) {
        alert(error.message);
    }
}

// Delete item
async function deleteItem(itemId) {
    if (!confirm('Are you sure you want to delete this item?')) {
        return;
    }
    
    try {
        const response = await fetch(`${API_BASE}/items/${itemId}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${currentToken}`
            }
        });
        
        if (!response.ok) {
            throw new Error('Failed to delete item');
        }
        
        loadItems();
        alert('Item deleted successfully!');
        
    } catch (error) {
        alert(error.message);
    }
}

// View item history
async function viewItemHistory(itemId) {
    showSection('history');
    document.getElementById('history-item-id').value = itemId;
    loadHistory();
}

// Load history
async function loadHistory() {
    if (!currentToken) {
        alert('Please login first');
        return;
    }
    
    const itemId = document.getElementById('history-item-id').value;
    const user = document.getElementById('history-user').value;
    const action = document.getElementById('history-action').value;
    
    let url = `${API_BASE}/items/${itemId}/history?limit=50`;
    if (user) url += `&changed_by=${user}`;
    if (action) url += `&action=${action}`;
    
    try {
        const response = await fetch(url, {
            headers: {
                'Authorization': `Bearer ${currentToken}`
            }
        });
        
        if (!response.ok) {
            throw new Error('Failed to load history');
        }
        
        const history = await response.json();
        renderHistory(history);
        
    } catch (error) {
        alert(error.message);
    }
}

// Render history table
function renderHistory(history) {
    const tbody = document.getElementById('history-table-body');
    tbody.innerHTML = '';
    
    if (history.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="text-center">No history found</td></tr>';
        return;
    }
    
    history.forEach(record => {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td>${new Date(record.changed_at).toLocaleString()}</td>
            <td>${record.item_id}</td>
            <td>
                <span class="badge ${getActionBadgeClass(record.action)}">
                    ${record.action}
                </span>
            </td>
            <td>${record.changed_by}</td>
            <td>${record.changes ? Object.keys(JSON.parse(record.changes)).length : 0} changes</td>
            <td>
                <button class="btn btn-sm btn-outline-info" onclick="showHistoryDetails(${record.id})">
                    <i class="bi bi-eye"></i> View
                </button>
            </td>
        `;
        tbody.appendChild(row);
    });
}

// Show history details
async function showHistoryDetails(historyId) {
    try {
        // Get history details
        const response = await fetch(`${API_BASE}/history/${historyId}/diff`, {
            headers: {
                'Authorization': `Bearer ${currentToken}`
            }
        });
        
        if (!response.ok) {
            throw new Error('Failed to load history details');
        }
        
        const changes = await response.json();
        
        // Get full history record
        const itemId = document.getElementById('history-item-id').value;
        const historyResponse = await fetch(`${API_BASE}/items/${itemId}/history?limit=50`, {
            headers: {
                'Authorization': `Bearer ${currentToken}`
            }
        });
        
        const history = await historyResponse.json();
        const record = history.find(h => h.id === historyId);
        
        // Populate modal
        document.getElementById('old-data').textContent = 
            record.old_data ? JSON.stringify(JSON.parse(record.old_data), null, 2) : 'No data';
        document.getElementById('new-data').textContent = 
            record.new_data ? JSON.stringify(JSON.parse(record.new_data), null, 2) : 'No data';
        
        const changesTable = document.getElementById('changes-table');
        changesTable.innerHTML = '';
        
        changes.forEach(change => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td><strong>${change.field}</strong></td>
                <td>${change.old !== null ? change.old : '<em>null</em>'}</td>
                <td>${change.new !== null ? change.new : '<em>null</em>'}</td>
            `;
            changesTable.appendChild(row);
        });
        
        // Show modal
        new bootstrap.Modal(document.getElementById('historyDetailsModal')).show();
        
    } catch (error) {
        alert(error.message);
    }
}

// Export history
async function exportHistory() {
    if (!currentToken) {
        alert('Please login first');
        return;
    }
    
    const itemId = document.getElementById('export-item-id').value;
    if (!itemId) {
        alert('Please enter an item ID');
        return;
    }
    
    try {
        const response = await fetch(`${API_BASE}/items/${itemId}/history/export`, {
            headers: {
                'Authorization': `Bearer ${currentToken}`
            }
        });
        
        if (!response.ok) {
            throw new Error('Failed to export history');
        }
        
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `history_item_${itemId}.csv`;
        document.body.appendChild(a);
        a.click();
        window.URL.revokeObjectURL(url);
        document.body.removeChild(a);
        
        alert('Export completed successfully!');
        
    } catch (error) {
        alert(error.message);
    }
}

// Utility functions
function getActionBadgeClass(action) {
    switch(action) {
        case 'CREATE': return 'bg-success';
        case 'UPDATE': return 'bg-warning';
        case 'DELETE': return 'bg-danger';
        default: return 'bg-secondary';
    }
}

function showAddItemModal() {
    document.getElementById('add-item-form').reset();
    new bootstrap.Modal(document.getElementById('addItemModal')).show();
}

function showEditItemModal(itemId) {
    // In a real app, you would fetch the item data first
    document.getElementById('edit-item-id').value = itemId;
    // Populate form with item data...
    new bootstrap.Modal(document.getElementById('editItemModal')).show();
}

// Initialize
document.addEventListener('DOMContentLoaded', function() {
    // Check if we have a saved token
    const savedToken = localStorage.getItem('token');
    if (savedToken) {
        // Verify token and set user
        // (In real app, you'd verify with backend)
    }
});
