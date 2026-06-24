import base64
import json
import time


def base64_url_encode(data: bytes) -> str:
    """Helper to create URL-safe base64 strings without padding, as specified by JWT."""
    return base64.urlsafe_b64encode(data).decode("utf-8").rstrip("=")


def _format_private_key(pem_str: str) -> str:
    """Formats a PEM string to ensure proper newlines and headers."""
    raw_key = pem_str.strip()
    if "\n" not in raw_key and " " in raw_key:
        header = "-----BEGIN RSA PRIVATE KEY-----"
        footer = "-----END RSA PRIVATE KEY-----"
        if "BEGIN PRIVATE KEY" in raw_key and "RSA" not in raw_key:
            header = "-----BEGIN PRIVATE KEY-----"
            footer = "-----END PRIVATE KEY-----"
        body_payload = raw_key.replace(header, "").replace(footer, "").strip()
        clean_body_tokens = body_payload.split()
        reconstructed_body = "\n".join(clean_body_tokens)
        return f"{header}\n{reconstructed_body}\n{footer}"
    else:
        return raw_key.replace("\r\n", "\n")


def generate_github_jwt(client_id: str, private_key: str) -> str:
    """Mints a short-lived JWT signed with RS256 using the GitHub App Private Key."""
    from cryptography.hazmat.primitives import hashes
    from cryptography.hazmat.primitives.asymmetric import padding
    from cryptography.hazmat.primitives.serialization import load_pem_private_key

    # 1. JWT Headers & Claims required by GitHub
    now = int(time.time())
    header = {"alg": "RS256", "typ": "JWT"}

    # GitHub tokens must be issued slightly in the past to compensate for server clock drift
    payload = {
        "iat": now - 60,  # Issued at
        "exp": now + (10 * 60),  # Expires in 10 minutes maximum
        "iss": client_id,  # Your GitHub App ID
    }

    # 2. Stringify and Base64 URL Encode segment inputs
    unsigned_header = base64_url_encode(json.dumps(header, separators=(",", ":")).encode("utf-8"))
    unsigned_payload = base64_url_encode(json.dumps(payload, separators=(",", ":")).encode("utf-8"))
    signing_input = f"{unsigned_header}.{unsigned_payload}".encode()

    # 3. Load the Multi-line RSA PEM key and Sign via RS256
    try:
        formatted_private_key = _format_private_key(private_key)
        private_key_obj = load_pem_private_key(formatted_private_key.encode("utf-8"), password=None)
        signature = private_key_obj.sign(signing_input, padding.PKCS1v15(), hashes.SHA256())
    except Exception as e:
        raise ValueError(f"Invalid Private Key formatting: {str(e)}") from e

    # 4. Form the complete JWT token string
    encoded_signature = base64_url_encode(signature)
    return f"{signing_input.decode('utf-8')}.{encoded_signature}"
