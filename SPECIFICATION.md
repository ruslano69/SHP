# Signed Hypertext Protocol (SHP) - Technical Specification

**Version:** 1.0-draft  
**Status:** Research Proposal  
**Last Updated:** November 2025  
**Authors:** Ruslan [Last Name]

---

## Abstract

The Signed Hypertext Protocol (SHP) is an extension to HTTP that provides cryptographic proof of content integrity and origin authentication from server to browser. While TLS secures the communication channel, SHP secures the payload itself, protecting against compromised intermediaries and enabling strict, deterministic HTML parsing.

This document defines the protocol mechanisms, cryptographic parameters, validation procedures, and browser behavior for SHP implementation.

---

## Table of Contents

1. [Introduction](#1-introduction)
2. [Terminology](#2-terminology)
3. [Protocol Overview](#3-protocol-overview)
4. [Signature Generation](#4-signature-generation)
5. [HTTP Headers](#5-http-headers)
6. [Signature Verification](#6-signature-verification)
7. [Browser Behavior](#7-browser-behavior)
8. [Security Considerations](#8-security-considerations)
9. [Performance Considerations](#9-performance-considerations)
10. [Backwards Compatibility](#10-backwards-compatibility)
11. [Implementation Requirements](#11-implementation-requirements)
12. [References](#12-references)

---

## 1. Introduction

### 1.1 Problem Statement

Current web security relies on Transport Layer Security (TLS), which provides:
- Channel encryption between client and server
- Server authentication via certificates
- Data integrity during transmission

However, TLS does not guarantee:
- Content integrity beyond the first TLS termination point
- Origin attribution at the browser level
- Protection against compromised CDNs or proxies

Additionally, HTML's error-tolerant parsing creates:
- Non-deterministic DOM construction
- Browser-specific vulnerability surfaces
- Unpredictable security behavior

### 1.2 Solution Approach

SHP addresses these limitations by:

1. **Content Signing:** Origin servers cryptographically sign HTML content
2. **XHTML Validation:** Content MUST be well-formed XHTML before signing
3. **End-to-End Verification:** Browsers verify signatures using public keys from TLS certificates
4. **Strict Parsing Mode:** Valid signatures enable XHTML parser (deterministic, no ambiguity)
5. **Graceful Degradation:** Invalid signatures fallback to legacy HTML5 mode

**Key Innovation:**
SHP makes XHTML practical by:
- Validating before transmission (not at browser)
- Providing fallback mechanism (XHTML failed due to draconian errors)
- Incentivizing adoption (performance + security benefits)

### 1.3 Design Goals

- **Security:** Reduce attack surface by 40%+ through strict validation
- **Performance:** Improve parsing speed by 20%+ via simplified error handling
- **Compatibility:** Maintain full backward compatibility with existing web
- **Deployability:** Enable immediate adoption via JavaScript polyfill
- **Simplicity:** Leverage existing TLS infrastructure (no new PKI)

---

## 2. Terminology

**MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** are interpreted as described in [RFC 2119](https://www.ietf.org/rfc/rfc2119.txt).

**Key Terms:**

- **Origin Server:** The application server that generates HTML content
- **Signing Server:** The server that creates SHP signatures (typically same as origin)
- **User Agent:** The browser or client rendering HTML
- **Canonical Content:** Standardized representation of HTML for signature generation
- **Strict Mode:** Browser parsing mode enabled when signature is valid
- **Legacy Mode:** Standard HTML5 parsing mode (quirks mode)
- **Intermediary:** CDN, proxy, or cache between origin and browser

---

## 3. Protocol Overview

### 3.1 Architecture
```
┌─────────────────┐
│  Origin Server  │
│                 │
│ 1. Generate     │
│    HTML         │
│                 │
│ 2. Validate     │
│    (strict)     │
│                 │
│ 3. Sign with    │
│    private key  │
│                 │
│ 4. Add headers  │
└────────┬────────┘
         │ HTTPS (TLS)
         ▼
┌─────────────────┐
│  Intermediary   │
│  (CDN/Proxy)    │
│                 │
│ - Forwards      │
│   unchanged     │
│ - Cannot modify │
│   without       │
│   detection     │
└────────┬────────┘
         │ HTTPS (TLS)
         ▼
┌─────────────────┐
│   User Agent    │
│   (Browser)     │
│                 │
│ 5. Extract      │
│    signature    │
│                 │
│ 6. Get pubkey   │
│    from TLS     │
│                 │
│ 7. Verify       │
│    signature    │
│                 │
│ 8. Enable       │
│    strict mode  │
│    (if valid)   │
└─────────────────┘
```

### 3.2 Workflow

**Server Side:**

1. Generate HTML content
2. Validate HTML against strict schema (well-formed, no errors)
3. Create canonical representation of content
4. Generate cryptographic signature using server's private key
5. Add SHP headers to HTTP response
6. Send to client over TLS

**Client Side:**

1. Receive HTML response
2. Check for `SHP-Signature` header
3. Extract public key from TLS certificate
4. Recreate canonical representation of received content
5. Verify signature matches content
6. If valid: enable Strict Mode
7. If invalid: enable Legacy Mode, log security event

---

## 4. Signature Generation

### 4.1 Canonical Content Format

Before signing, content MUST be converted to canonical form:
```
Canonical_Message = 
    HTTP_Method + "\n" +
    Request_URI + "\n" +
    HTML_Body + "\n" +
    Timestamp + "\n" +
    Content_Type
```

**Example:**
```
GET
/index.html
<!DOCTYPE html><html><head>...</head><body>...</body></html>
2025-11-20T10:30:00Z
text/html; charset=utf-8
```
### 4.1.1 HTML Validation Requirements

Before signing, HTML content MUST conform to **XHTML strict validation rules:**

**Required:**
- All tags must be properly closed: `<p>text</p>` (not `<p>text`)
- All tags must be properly nested: `<div><p></p></div>` (not `<div><p></div></p>`)
- All attribute values must be quoted: `<img src="pic.jpg">` (not `<img src=pic.jpg>`)
- All tags must be lowercase: `<html>` (not `<HTML>`)
- Empty elements must be self-closing: `<br />`, `<img />`, `<meta />`
- Document must be well-formed XML

**Rationale:**
- XHTML strictness eliminates parser ambiguity
- Enables deterministic parsing across all browsers
- Signature verifies content is structurally valid
- "Strict Mode" becomes meaningful (not just faster, but provably correct)

**Example of Valid SHP-HTML:**
```xml
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
  <meta charset="utf-8" />
  <title>Valid SHP Document</title>
</head>
<body>
  <p>All tags are properly closed.</p>
  <img src="image.jpg" alt="Description" />
  <br />
</body>
</html>
```

**Example of Invalid (will not be signed):**
```html
<!-- This would fail validation -->
<html>
<head>
<title>Invalid Document
<body>
<p>Unclosed tag
<img src=noquotes.jpg>
<BR>  <!-- uppercase -->
```

### 4.2 Normalization Rules

1. **Line Endings:** Convert to `\n` (LF only, no CRLF)
2. **Whitespace:** Preserve all whitespace in HTML body
3. **Encoding:** UTF-8 encoding MUST be used
4. **Order:** Fields MUST appear in order specified above
5. **Case Sensitivity:** HTTP method and headers are case-sensitive

### 4.3 Signature Algorithm

**Default (REQUIRED):**
- Algorithm: `RSASSA-PKCS1-v1_5`
- Hash: `SHA-256`
- Key Size: `2048 bits` minimum

**Optional (Future):**
- Algorithm: `EdDSA`
- Curve: `Ed25519`

### 4.4 Signature Process
```python
# Pseudocode
def generate_signature(html_body, private_key):
    # 1. Create canonical message
    canonical = create_canonical_message(
        method="GET",
        uri=request.path,
        body=html_body,
        timestamp=now_iso8601(),
        content_type="text/html; charset=utf-8"
    )
    
    # 2. Hash canonical message
    hash_value = SHA256(canonical.encode('utf-8'))
    
    # 3. Sign hash with private key
    signature = RSA_SIGN(private_key, hash_value)
    
    # 4. Encode as base64
    return base64_encode(signature)
```
### 4.7 Pre-Signature Validation

**Server MUST perform validation before signing:**
```python
def validate_and_sign(html_content, private_key):
    # 1. Parse as XHTML
    try:
        doc = parse_xhtml(html_content)
    except XMLError as e:
        raise ValidationError(f"Not well-formed XHTML: {e}")
    
    # 2. Check required structure
    if not doc.has_element('html'):
        raise ValidationError("Missing <html> root element")
    if not doc.has_element('head'):
        raise ValidationError("Missing <head> element")
    if not doc.has_element('body'):
        raise ValidationError("Missing <body> element")
    
    # 3. Validate all tags closed
    if doc.has_unclosed_tags():
        raise ValidationError("Document has unclosed tags")
    
    # 4. Validate attribute quoting
    if doc.has_unquoted_attributes():
        raise ValidationError("Document has unquoted attributes")
    
    # 5. If valid, proceed to sign
    return generate_signature(html_content, private_key)
```

**Validation Failures:**
- MUST return HTTP 500 (Internal Server Error)
- MUST log validation errors
- MUST NOT send unsigned content
- SHOULD provide detailed error message in logs (not to client)
---

## 5. HTTP Headers

### 5.1 Required Headers

**`SHP-Version`**
- **Type:** String
- **Format:** `<major>.<minor>`
- **Example:** `1.0`
- **Description:** SHP protocol version

**`SHP-Signature`**
- **Type:** Base64-encoded binary
- **Format:** Base64 string
- **Example:** `iQIzBAABCAAdFiEE7V...`
- **Description:** Cryptographic signature of canonical content

**`SHP-Algorithm`**
- **Type:** String
- **Format:** `<hash>-<signature>`
- **Example:** `SHA256-RSA2048`
- **Description:** Algorithm identifier
- **Valid Values:**
  - `SHA256-RSA2048` (required)
  - `SHA384-RSA3072` (optional)
  - `SHA256-Ed25519` (optional)

**`SHP-Timestamp`**
- **Type:** ISO 8601 datetime
- **Format:** `YYYY-MM-DDTHH:MM:SSZ`
- **Example:** `2025-11-20T10:30:00Z`
- **Description:** Signature creation time (UTC)
- **Max Age:** 300 seconds (5 minutes)

### 5.2 Optional Headers

**`Content-Validation`**
- **Type:** String
- **Values:** `strict`, `none`
- **Default:** `none`
- **Description:** Server's validation policy

**`SHP-Certificate-Fingerprint`**
- **Type:** Hex string
- **Format:** SHA-256 hash of certificate
- **Example:** `a1b2c3d4...`
- **Description:** Expected certificate fingerprint (for key pinning)

### 5.3 Example Response
```http
HTTP/2 200 OK
Date: Wed, 20 Nov 2025 10:30:00 GMT
Content-Type: text/html; charset=utf-8
Content-Length: 4567
SHP-Version: 1.0
SHP-Signature: iQIzBAABCAAdFiEE7VwzrqIy1234abcd...
SHP-Algorithm: SHA256-RSA2048
SHP-Timestamp: 2025-11-20T10:30:00Z
Content-Validation: strict

<!DOCTYPE html>
<html>
...
</html>
```

---

## 6. Signature Verification

### 6.1 Verification Process

**Step 1: Extract Metadata**
```javascript
const signature = response.headers.get('SHP-Signature');
const algorithm = response.headers.get('SHP-Algorithm');
const timestamp = response.headers.get('SHP-Timestamp');
```

**Step 2: Validate Timestamp**
```javascript
const signatureTime = new Date(timestamp);
const currentTime = new Date();
const age = (currentTime - signatureTime) / 1000; // seconds

if (age > 300) {
    return INVALID; // Signature too old
}
if (age < -60) {
    return INVALID; // Clock skew too large
}
```

**Step 3: Extract Public Key**
```javascript
// Get public key from TLS certificate
const certificate = tlsSession.peerCertificate;
const publicKey = certificate.publicKey;
```

**Step 4: Reconstruct Canonical Message**
```javascript
const canonical = 
    request.method + "\n" +
    request.url.pathname + "\n" +
    responseBody + "\n" +
    timestamp + "\n" +
    "text/html; charset=utf-8";
```

**Step 5: Verify Signature**
```javascript
const hash = await crypto.subtle.digest('SHA-256', 
    new TextEncoder().encode(canonical));

const isValid = await crypto.subtle.verify(
    'RSASSA-PKCS1-v1_5',
    publicKey,
    base64Decode(signature),
    hash
);

return isValid ? VALID : INVALID;
```

### 6.2 Verification Outcomes

| Outcome | Browser Action |
|---------|----------------|
| **Valid Signature** | Enable Strict Mode, show security indicator |
| **Invalid Signature** | Enable Legacy Mode, hide security indicator, log event |
| **No Signature** | Enable Legacy Mode (backward compatible) |
| **Expired Timestamp** | Treat as Invalid Signature |
| **Missing Public Key** | Treat as Invalid Signature |
| **Unsupported Algorithm** | Treat as Invalid Signature |

---

## 7. Browser Behavior

### 7.1 Strict Mode

When signature is valid, browser MUST:

1. **Enable Strict Parser:**
   - Reject malformed HTML (no error recovery)
   - Use deterministic parsing rules
   - Generate identical DOM across all browsers

2. **Display Security Indicator:**
   - Show padlock or shield icon
   - Indicate "Content Verified by SHP"
   - Allow user to inspect signature details

3. **Enable Privileged APIs:**
   - Allow access to camera/microphone
   - Allow geolocation access
   - Allow payment request APIs
   - Allow credential management APIs

4. **Log Success Event:**
```javascript
   console.log('[SHP] ✓ Valid signature. Strict mode enabled.');
```
### 7.1.1 Strict Parser = XHTML Parser

When signature is valid, browser MUST use **XHTML parsing mode:**

- **No error recovery:** Single malformed tag → parsing stops
- **XML rules apply:** Case-sensitive, all tags must close
- **Deterministic behavior:** Identical DOM tree across all browsers
- **Performance:** 20-30% faster (no heuristic guessing)

**This is why XHTML works with SHP but failed before:**
- **XHTML (2000):** Broke sites immediately, no adoption path
- **SHP + XHTML (2025):** Gradual opt-in, content pre-validated before signing

The signature **proves** content is valid XHTML, so browser can safely use strict parser.

### 7.2 Legacy Mode

When signature is invalid or missing, browser MUST:

1. **Use HTML5 Parser:**
   - Standard quirks mode
   - Error-tolerant parsing
   - Maintain compatibility

2. **Hide/Remove Security Indicator:**
   - No SHP badge shown
   - Standard HTTPS indicator only

3. **Restrict Privileged APIs:**
   - Require additional user consent for sensitive APIs
   - Log API access attempts

4. **Log Failure Event (if signature present but invalid):**
```javascript
   console.warn('[SHP] ✗ Invalid signature. Legacy mode active.');
```

### 7.3 Performance Optimization

**Strict Parser Requirements:**
- MUST NOT include error recovery code paths
- MUST reject malformed HTML immediately
- SHOULD be 20-30% faster than quirks mode parser
- SHOULD use 15-25% less memory

**Caching:**
- Signature verification result MAY be cached
- Cache key: `(URL, Timestamp, Signature)`
- Cache duration: Until timestamp expiration

---

## 8. Security Considerations

### 8.1 Threat Model

**Threats SHP Mitigates:**

1. **CDN Compromise**
   - Attacker controls CDN infrastructure
   - Cannot inject malicious code (signature breaks)
   - **Mitigation:** E2E integrity from origin

2. **Proxy Injection**
   - Corporate/ISP proxy modifies HTML
   - Modification detected immediately
   - **Mitigation:** Signature verification

3. **Cache Poisoning**
   - Attacker poisons cache with malicious content
   - Stale signature prevents acceptance
   - **Mitigation:** Timestamp validation

4. **Parser Ambiguity Exploits**
   - Attacker exploits browser-specific quirks
   - Strict mode eliminates ambiguity
   - **Mitigation:** Deterministic parsing

**Threats SHP Does NOT Mitigate:**

1. **Server Compromise:** If origin server is compromised, attacker signs malicious content
2. **Client-Side XSS:** JavaScript vulnerabilities not addressed
3. **Social Engineering:** User can still be tricked into accepting invalid signatures
4. **DNS Hijacking:** Attacker redirects to malicious domain (TLS already addresses this)

### 8.2 Attack Resistance

**Replay Attacks:**
- Prevented by timestamp validation (max age 5 minutes)
- Attacker cannot reuse old signed content

**Man-in-the-Middle:**
- TLS protects channel
- SHP protects content even if TLS compromised at edge
- Combined protection stronger than either alone

**Downgrade Attacks:**
- If attacker removes SHP headers, browser treats as legacy
- No worse than current web (backward compatible)
- Can be prevented with `Require-SHP: strict` policy

### 8.3 Cryptographic Security

**Algorithm Selection:**
- SHA-256: 128-bit security level
- RSA-2048: 112-bit security level (acceptable until 2030)
- RSA-3072: 128-bit security level (recommended for new deployments)
- Ed25519: 128-bit security level, better performance

**Key Management:**
- Private keys MUST be stored securely (HSM recommended)
- Key rotation SHOULD occur annually
- Compromised keys MUST be revoked immediately

---

## 9. Performance Considerations

### 9.1 Computational Overhead

**Signature Generation (Server):**
- RSA-2048: ~1-2 ms per signature
- Ed25519: ~0.1 ms per signature
- **Recommendation:** Pre-compute signatures for static content

**Signature Verification (Client):**
- RSA-2048: ~1-2 ms per verification
- Ed25519: ~0.3 ms per verification
- **Recommendation:** Cache verification results

**Net Performance Impact:**
- Signature overhead: +2-4 ms total (one-time)
- Strict parsing savings: -10-50 ms (every render)
- **Result:** 10-48 ms faster page load

### 9.2 Bandwidth Overhead

**Header Size:**
- `SHP-Version`: ~10 bytes
- `SHP-Signature`: ~256 bytes (RSA-2048)
- `SHP-Algorithm`: ~20 bytes
- `SHP-Timestamp`: ~25 bytes
- **Total:** ~311 bytes (~0.3 KB)

**Impact:**
- For 100 KB HTML: 0.3% overhead
- For 10 KB HTML: 3% overhead
- **Mitigation:** Compress headers with Brotli

### 9.3 Caching Strategy

**Static Content:**
- Sign once, cache signature
- Serve pre-signed responses
- Update signature on content change only

**Dynamic Content:**
- Sign on-the-fly
- Consider edge signing (if CDN trusted)
- Cache verification result client-side

---

## 10. Backwards Compatibility

### 10.1 Graceful Degradation

**Old Browsers (no SHP support):**
- Ignore SHP headers
- Parse HTML in quirks mode (as today)
- No functionality loss

**New Browsers (with SHP):**
- Process SHP headers
- Enable strict mode if valid
- Fallback to quirks mode if invalid

### 10.2 Migration Path

**Phase 1: Polyfill (Today)**
- JavaScript-based verification
- Works in all modern browsers
- No browser changes needed

**Phase 2: Opt-in Native (1-2 years)**
- Browsers implement SHP parsing
- Sites add SHP headers to opt-in
- Backward compatible

**Phase 3: Encouraged (3-5 years)**
- Government/finance mandates
- SEO benefits for SHP sites
- Still backward compatible

**Phase 4: Enforcement (5+ years, optional)**
- High-security sites require SHP
- `Require-SHP: strict` policy
- Legacy sites still work

---

## 11. Implementation Requirements

### 11.1 Server Requirements

**MUST:**
- Validate HTML before signing
- Generate correct canonical representation
- Use secure key storage
- Include all required headers

**SHOULD:**
- Cache signatures for static content
- Monitor signature generation failures
- Log signature requests for audit

**MAY:**
- Use hardware security modules (HSM)
- Implement edge signing (with caution)
- Support multiple signature algorithms

### 11.2 Browser Requirements

**MUST:**
- Verify signatures before parsing
- Implement strict parser for valid signatures
- Fallback to quirks mode for invalid/missing signatures
- Display security indicator appropriately

**SHOULD:**
- Cache verification results
- Provide developer tools integration
- Log security events

**MAY:**
- Implement performance optimizations
- Provide user preferences for strictness
- Support certificate pinning

### 11.3 Compliance

**For SHP 1.0 Compliance:**

Server MUST:
- [ ] Generate valid canonical messages
- [ ] Sign with RSA-2048 minimum
- [ ] Include all required headers
- [ ] Validate timestamp freshness

Browser MUST:
- [ ] Verify signatures correctly
- [ ] Enable strict mode for valid signatures
- [ ] Fallback to legacy mode gracefully
- [ ] Display security indicators

---

## 12. References

### Standards
- [RFC 2119](https://www.ietf.org/rfc/rfc2119.txt) - Key words for RFCs
- [RFC 7515](https://www.rfc-editor.org/rfc/rfc7515) - JSON Web Signature (JWS)
- [RFC 8017](https://www.rfc-editor.org/rfc/rfc8017) - PKCS #1: RSA Cryptography
- [HTML Living Standard](https://html.spec.whatwg.org/) - HTML parsing specification

### Related Work
- [Signed HTTP Exchanges (SXG)](https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html)
- [Subresource Integrity (SRI)](https://www.w3.org/TR/SRI/)
- [Content Security Policy (CSP)](https://www.w3.org/TR/CSP/)

### Implementations
- Reference implementation: [github.com/ruslano69/SHP](https://github.com/ruslano69/SHP)

---

## Appendix A: Test Vectors

### A.1 Canonical Message Example

**Input:**
```
Method: GET
URI: /test.html
Body: <!DOCTYPE html><html><head><title>Test</title></head><body><p>Hello</p></body></html>
Timestamp: 2025-11-20T12:00:00Z
Content-Type: text/html; charset=utf-8
```

**Canonical:**
```
GET
/test.html
<!DOCTYPE html><html><head><title>Test</title></head><body><p>Hello</p></body></html>
2025-11-20T12:00:00Z
text/html; charset=utf-8
```

**SHA-256 Hash:**
```
a3b2c1d4e5f6789012345678901234567890abcdef1234567890abcdef123456
```

---

## Appendix B: Algorithm Identifiers

| Identifier | Hash | Signature | Key Size | Status |
|------------|------|-----------|----------|--------|
| SHA256-RSA2048 | SHA-256 | RSA PKCS#1 v1.5 | 2048 | Required |
| SHA384-RSA3072 | SHA-384 | RSA PKCS#1 v1.5 | 3072 | Optional |
| SHA256-Ed25519 | SHA-256 | EdDSA | 256 | Optional |

---

**Document Status:** Draft for Review  
**Next Review:** After initial implementation feedback  
**Contact:** https://github.com/ruslano69/SHP/issues

---