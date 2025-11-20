/**
 * SHP Protocol - Browser Polyfill (Proof of Concept)
 * Validates HTML content signature using Web Crypto API
 * 
 * Usage:
 *   <meta name="shp-signature" content="base64-signature">
 *   <meta name="shp-algorithm" content="SHA-256">
 *   <meta name="shp-pubkey" content="base64-public-key">
 *   <script src="shp-verify.js"></script>
 */

(async function() {
    'use strict';

    const SHP = {
        version: '1.0.0-alpha',
        
        /**
         * Extract metadata from page
         */
        extractMetadata() {
            const getMeta = (name) => {
                const el = document.querySelector(`meta[name="${name}"]`);
                return el ? el.getAttribute('content') : null;
            };

            return {
                signature: getMeta('shp-signature'),
                algorithm: getMeta('shp-algorithm') || 'SHA-256',
                pubkey: getMeta('shp-pubkey'),
                timestamp: getMeta('shp-timestamp')
            };
        },

        /**
         * Convert base64 to ArrayBuffer
         */
        base64ToArrayBuffer(base64) {
            const binary = atob(base64);
            const bytes = new Uint8Array(binary.length);
            for (let i = 0; i < binary.length; i++) {
                bytes[i] = binary.charCodeAt(i);
            }
            return bytes.buffer;
        },

        /**
         * Import RSA public key from base64
         */
        async importPublicKey(base64Key) {
            try {
                const keyData = this.base64ToArrayBuffer(base64Key);
                return await crypto.subtle.importKey(
                    'spki',
                    keyData,
                    {
                        name: 'RSASSA-PKCS1-v1_5',
                        hash: 'SHA-256'
                    },
                    false,
                    ['verify']
                );
            } catch (error) {
                console.error('[SHP] Failed to import public key:', error);
                return null;
            }
        },

        /**
         * Create canonical representation of content
         */
        createCanonicalContent() {
            // Simplified: hash entire HTML (in production, would be more sophisticated)
            const html = document.documentElement.outerHTML;
            // Remove script tags to avoid chicken-egg problem
            const cleaned = html.replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '');
            return new TextEncoder().encode(cleaned);
        },

        /**
         * Verify signature
         */
        async verifySignature(publicKey, signature, content) {
            try {
                const signatureBuffer = this.base64ToArrayBuffer(signature);
                const contentBuffer = content;

                const isValid = await crypto.subtle.verify(
                    'RSASSA-PKCS1-v1_5',
                    publicKey,
                    signatureBuffer,
                    contentBuffer
                );

                return isValid;
            } catch (error) {
                console.error('[SHP] Signature verification failed:', error);
                return false;
            }
        },

        /**
         * Display security indicator
         */
        showIndicator(valid) {
            const indicator = document.createElement('div');
            indicator.id = 'shp-indicator';
            indicator.style.cssText = `
                position: fixed;
                top: 10px;
                right: 10px;
                padding: 10px 20px;
                border-radius: 5px;
                font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
                font-size: 14px;
                font-weight: 600;
                z-index: 999999;
                box-shadow: 0 2px 10px rgba(0,0,0,0.2);
                cursor: pointer;
                transition: opacity 0.3s;
            `;

            if (valid) {
                indicator.style.background = '#10b981';
                indicator.style.color = '#fff';
                indicator.innerHTML = 'ðŸ”’ Verified by SHP';
                indicator.title = 'Content signature is valid. Strict mode enabled.';
            } else {
                indicator.style.background = '#ef4444';
                indicator.style.color = '#fff';
                indicator.innerHTML = 'âš ï¸ Invalid Signature';
                indicator.title = 'Content signature is invalid. Legacy mode active.';
            }

            // Auto-hide after 5 seconds
            setTimeout(() => {
                indicator.style.opacity = '0';
                setTimeout(() => indicator.remove(), 300);
            }, 5000);

            // Click to dismiss
            indicator.addEventListener('click', () => {
                indicator.style.opacity = '0';
                setTimeout(() => indicator.remove(), 300);
            });

            document.body.appendChild(indicator);
        },

        /**
         * Log security event
         */
        logSecurityEvent(valid, details) {
            const event = {
                timestamp: new Date().toISOString(),
                protocol: 'SHP',
                version: this.version,
                valid: valid,
                url: window.location.href,
                details: details
            };

            console.log('[SHP] Security Event:', event);

            // In production: send to monitoring system
            // fetch('/api/security-events', { method: 'POST', body: JSON.stringify(event) });
        },

        /**
         * Main validation routine
         */
        async validate() {
            console.log('[SHP] Starting validation...');

            const metadata = this.extractMetadata();

            // Check if SHP metadata exists
            if (!metadata.signature || !metadata.pubkey) {
                console.log('[SHP] No signature found. Operating in legacy mode.');
                return;
            }

            console.log('[SHP] Signature found. Verifying...');

            // Import public key
            const publicKey = await this.importPublicKey(metadata.pubkey);
            if (!publicKey) {
                this.showIndicator(false);
                this.logSecurityEvent(false, { error: 'Failed to import public key' });
                return;
            }

            // Create canonical content
            const content = this.createCanonicalContent();

            // Verify signature
            const isValid = await this.verifySignature(publicKey, metadata.signature, content);

            // Show result
            this.showIndicator(isValid);
            this.logSecurityEvent(isValid, {
                algorithm: metadata.algorithm,
                timestamp: metadata.timestamp
            });

            if (isValid) {
                console.log('[SHP] âœ“ Signature valid. Strict mode enabled.');
                document.documentElement.setAttribute('data-shp-mode', 'strict');
            } else {
                console.log('[SHP] âœ— Signature invalid. Legacy mode active.');
                document.documentElement.setAttribute('data-shp-mode', 'legacy');
            }
        }
    };

    // Auto-run when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', () => SHP.validate());
    } else {
        SHP.validate();
    }

    // Expose API
    window.SHP = SHP;

})();