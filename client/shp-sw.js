/**
 * SHP Service Worker v2.0 - Raw Bytes Verification
 * Version: 2.0.0
 * Description: No parsing required - verify raw bytes directly!
 * 
 * This revolutionary approach eliminates HTML/DOM parsing entirely.
 * Service Worker receives signed XHTML, verifies the signature against raw bytes,
 * and if valid, passes to browser for strict XHTML parsing.
 */

// SHP Service Worker
self.addEventListener('install', (event) => {
    console.log('[SHP] Service Worker installing...');
    event.waitUntil(self.skipWaiting());
});

self.addEventListener('activate', (event) => {
    console.log('[SHP] Service Worker activated');
    event.waitUntil(self.clients.claim());
});

// SHP Configuration
const SHP_CONFIG = {
    // Public keys for verification (in production, get from secure source)
    PUBLIC_KEYS: [
        // Generated RSA-2048 public key (base64 SPKI format)
        'MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAyZks+JGbZJkGz4euEgZl8fEr/hcDsGBY+VI1HJioeMRC2NAeSl+VW7l3TewCaDVqJOCQF67Az60scoz9urp9/6m3mG5GUyoDS/pMmn1Qetw9XZlQTBuPFEWCkndIRz1W0zK/Grqpv6cv30PYNGgSm/EQaeiKw6XAH9gopM+dTVkcl3l6t6tKlpBtOGQXtQuSn0LMf3rhXnKvhVT1rrkVibH+qlmjNE95PyNL9M6yzERyc45FYeXCo2Wd6uDVMRnf08/Y2E1xd70QgIJ4ocB66Fu+ZOwFO3LFj0sL1GhC48LCeBAo5JdOkFgqGsSzVq7YJwUu2B/sJvYgse2DDAhw2wIDAQAB'
    ],
    // Cache settings
    CACHE_NAME: 'shp-cache-v1',
    // Debug mode
    DEBUG: true
};

// Main fetch handler
self.addEventListener('fetch', (event) => {
    event.respondWith(handleSHPRequest(event.request));
});

/**
 * Handle SHP requests - verify raw bytes, no parsing!
 */
async function handleSHPRequest(request) {
    try {
        // Fetch the response
        const response = await fetch(request);
        
        // Check for SHP signature
        const signature = response.headers.get('SHP-Signature');
        if (!signature) {
            // No SHP signature, pass through normally
            return response;
        }
        
        // Check SHP version
        const shpVersion = response.headers.get('SHP-Version') || '1.0';
        if (shpVersion !== '2.0') {
            console.warn('[SHP] Unsupported SHP version:', shpVersion);
            return response;
        }
        
        // Get response as raw bytes (critical!)
        const responseClone = response.clone();
        const htmlBytes = await responseClone.arrayBuffer();
        
        // Verify signature on RAW BYTES (no DOM parsing!)
        const isValid = await verifySignature(htmlBytes, signature, response.headers);
        
        if (isValid) {
            console.log('[SHP] ✅ Valid signature - strict XHTML mode');
            
            // Force strict XHTML parsing
            const strictHeaders = new Headers(response.headers);
            strictHeaders.set('Content-Type', 'application/xhtml+xml');
            strictHeaders.set('X-SHP-Verified', 'true');
            
            return new Response(htmlBytes, {
                status: response.status,
                statusText: response.statusText,
                headers: strictHeaders
            });
        } else {
            console.log('[SHP] ❌ INVALID signature - BLOCKING');
            return createBlockedResponse(request.url);
        }
        
    } catch (error) {
        console.error('[SHP] Error processing request:', error);
        // On error, block for security
        return createErrorResponse(request.url, error.message);
    }
}

/**
 * Verify signature against raw bytes (no DOM, no canonicalization!)
 */
async function verifySignature(htmlBytes, signatureB64, headers) {
    try {
        // Check timestamp (prevent replay attacks)
        const timestamp = headers.get('SHP-Timestamp');
        if (timestamp && !isTimestampValid(timestamp)) {
            console.warn('[SHP] Signature too old:', timestamp);
            return false;
        }
        
        // Get algorithm
        const algorithm = headers.get('SHP-Algorithm') || 'SHA256-RSA2048';
        if (algorithm !== 'SHA256-RSA2048') {
            console.warn('[SHP] Unsupported algorithm:', algorithm);
            return false;
        }
        
        // Import public key (in production, get from secure source)
        const publicKey = await getPublicKey();
        if (!publicKey) {
            console.error('[SHP] No public key available');
            return false;
        }
        
        // Decode signature
        const signature = base64ToArrayBuffer(signatureB64);
        
        // Verify signature against RAW BYTES
        const verified = await crypto.subtle.verify(
            {
                name: 'RSASSA-PKCS1-v1_5',
                hash: 'SHA-256'
            },
            publicKey,
            signature,
            htmlBytes
        );
        
        if (SHP_CONFIG.DEBUG) {
            console.log('[SHP] Signature verification:', {
                valid: verified,
                bytes: htmlBytes.byteLength,
                timestamp: timestamp,
                algorithm: algorithm
            });
        }
        
        return verified;
        
    } catch (error) {
        console.error('[SHP] Signature verification error:', error);
        return false;
    }
}

/**
 * Get public key for verification
 * In production, this should fetch from a secure source
 */
async function getPublicKey() {
    // For demo purposes, return null to indicate no key
    // In real implementation, fetch from server or use hardcoded key
    if (SHP_CONFIG.PUBLIC_KEYS.length === 0) {
        console.warn('[SHP] No public keys configured');
        return null;
    }
    
    // Import first available key
    const keyB64 = SHP_CONFIG.PUBLIC_KEYS[0];
    const binaryDer = base64ToArrayBuffer(keyB64);
    
    return await crypto.subtle.importKey(
        'spki',
        binaryDer,
        {
            name: 'RSASSA-PKCS1-v1_5',
            hash: 'SHA-256'
        },
        false,
        ['verify']
    );
}

/**
 * Check if timestamp is valid (not older than 5 minutes)
 */
function isTimestampValid(timestampStr) {
    try {
        const timestamp = new Date(timestampStr);
        const now = new Date();
        const ageMinutes = (now - timestamp) / 1000 / 60;
        
        return ageMinutes <= 5;
    } catch (error) {
        console.error('[SHP] Invalid timestamp format:', timestampStr);
        return false;
    }
}

/**
 * Create blocked response (signature invalid)
 */
function createBlockedResponse(url) {
    const warningPage = `
<!DOCTYPE html>
<html>
<head>
    <title>⚠️ SHP Security Warning</title>
    <style>
        body {
            background: #ff4444;
            color: white;
            font-family: Arial, sans-serif;
            padding: 50px;
            text-align: center;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            background: rgba(255,255,255,0.1);
            padding: 30px;
            border-radius: 10px;
        }
        h1 { font-size: 2.5em; margin-bottom: 20px; }
        h2 { font-size: 1.5em; margin-bottom: 20px; }
        .url { background: rgba(0,0,0,0.3); padding: 10px; border-radius: 5px; word-break: break-all; }
        .icon { font-size: 3em; margin: 20px 0; }
        .instructions {
            background: rgba(255,255,255,0.1);
            padding: 20px;
            border-radius: 5px;
            margin: 20px 0;
            text-align: left;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">⚠️</div>
        <h1>INVALID SHP SIGNATURE</h1>
        <h2>Content Integrity Compromised</h2>
        
        <p>The content at this URL has an invalid SHP signature:</p>
        <div class="url">${url}</div>
        
        <div class="instructions">
            <h3>What this means:</h3>
            <ul>
                <li>The content may have been modified by a proxy or CDN</li>
                <li>The signature does not match the actual content bytes</li>
                <li>This is a security protection measure</li>
            </ul>
            
            <h3>What you should do:</h3>
            <ul>
                <li>Do NOT trust this page content</li>
                <li>Contact the website administrator</li>
                <li>Try refreshing the page (the issue may be temporary)</li>
            </ul>
        </div>
        
        <p style="font-size: 0.9em; opacity: 0.8;">
            Protected by SHP (Signed Hypertext Protocol) v2.0
        </p>
    </div>
</body>
</html>`;

    return new Response(warningPage, {
        status: 403,
        statusText: 'Content Integrity Violation',
        headers: {
            'Content-Type': 'text/html',
            'X-SHP-Blocked': 'true'
        }
    });
}

/**
 * Create error response
 */
function createErrorResponse(url, errorMsg) {
    const errorPage = `
<!DOCTYPE html>
<html>
<head>
    <title>❌ SHP Error</title>
    <style>
        body {
            background: #ff9800;
            color: white;
            font-family: Arial, sans-serif;
            padding: 50px;
            text-align: center;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
        }
        h1 { font-size: 2.5em; }
        .error { background: rgba(0,0,0,0.2); padding: 15px; border-radius: 5px; margin: 20px 0; }
    </style>
</head>
<body>
    <h1>❌ SHP Verification Error</h1>
    <div class="error">
        <strong>Error:</strong> ${errorMsg}
    </div>
    <p><strong>URL:</strong> ${url}</p>
    <p>Please contact the website administrator.</p>
</body>
</html>`;

    return new Response(errorPage, {
        status: 500,
        statusText: 'SHP Verification Error',
        headers: {
            'Content-Type': 'text/html'
        }
    });
}

/**
 * Convert base64 to ArrayBuffer
 */
function base64ToArrayBuffer(base64) {
    const binaryString = atob(base64);
    const bytes = new Uint8Array(binaryString.length);
    for (let i = 0; i < binaryString.length; i++) {
        bytes[i] = binaryString.charCodeAt(i);
    }
    return bytes.buffer;
}

/**
 * Utility: Convert ArrayBuffer to base64
 */
function arrayBufferToBase64(buffer) {
    const bytes = new Uint8Array(buffer);
    let binary = '';
    for (let i = 0; i < bytes.byteLength; i++) {
        binary += String.fromCharCode(bytes[i]);
    }
    return btoa(binary);
}

// Log startup
console.log('[SHP] Service Worker loaded - Raw Bytes Verification v2.0');