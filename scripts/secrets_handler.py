import re
import json
import os
import sys

FILES_CONFIG = {
    "docker-compose.yml": [
        r"(DATABASE_PASSWORD=)([^\s-]+)()",
        r"(MYSQL_ROOT_PASSWORD=)([^\s-]+)()"
    ],
    "configs/config.yaml": [
        r'(password:\s*")([^"]*)(")',
        r'(oss_access_key:\s*")([^"]*)(")',
        r'(oss_secret_key:\s*")([^"]*)(")',
        r'(api_key:\s*")([^"]*)(")'
    ]
}

SECRETS_FILE = ".secrets.json"
MASK = "******"

def load_secrets():
    if os.path.exists(SECRETS_FILE):
        try:
            with open(SECRETS_FILE, 'r', encoding='utf-8') as f:
                return json.load(f)
        except Exception as e:
            print(f"Error loading secrets: {e}")
    return {}

def save_secrets(secrets):
    with open(SECRETS_FILE, 'w', encoding='utf-8') as f:
        json.dump(secrets, f, indent=4)

def mask():
    secrets = load_secrets()
    for filepath, patterns in FILES_CONFIG.items():
        if not os.path.exists(filepath):
            continue
        
        with open(filepath, 'r', encoding='utf-8') as f:
            content = f.read()
        
        file_secrets = []
        
        def mask_replacer(match):
            prefix = match.group(1)
            value = match.group(2)
            suffix = match.group(3)
            
            if value == MASK:
                return match.group(0)
            
            file_secrets.append(value)
            return f"{prefix}{MASK}{suffix}"

        new_content = content
        for pattern in patterns:
            new_content = re.sub(pattern, mask_replacer, new_content)

        if file_secrets:
            secrets[filepath] = file_secrets
            with open(filepath, 'w', encoding='utf-8') as f:
                f.write(new_content)
            print(f"Masked {len(file_secrets)} secrets in {filepath}")
    
    save_secrets(secrets)

def unmask():
    secrets = load_secrets()
    if not secrets:
        print("No secrets found to restore. Please make sure .secrets.json exists.")
        return

    for filepath, patterns in FILES_CONFIG.items():
        if not os.path.exists(filepath) or filepath not in secrets:
            continue
        
        with open(filepath, 'r', encoding='utf-8') as f:
            content = f.read()
        
        file_secrets = secrets[filepath]
        current_file_secrets = list(file_secrets)
        
        def unmask_replacer(match):
            if not current_file_secrets:
                return match.group(0)
            
            prefix = match.group(1)
            value = match.group(2)
            suffix = match.group(3)
            
            if value == MASK:
                orig_value = current_file_secrets.pop(0)
                return f"{prefix}{orig_value}{suffix}"
            return match.group(0)

        new_content = content
        for pattern in patterns:
            if "docker-compose.yml" in filepath:
                mask_search_pattern = pattern.replace(r"([^\s-]+)", f"({re.escape(MASK)})")
            else:
                mask_search_pattern = pattern.replace(r'([^"]*)', f"({re.escape(MASK)})")
            
            new_content = re.sub(mask_search_pattern, unmask_replacer, new_content)

        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(new_content)
        print(f"Unmasked {len(file_secrets) - len(current_file_secrets)} secrets in {filepath}")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python scripts/secrets_handler.py [mask|unmask]")
        sys.exit(1)
    
    cmd = sys.argv[1].lower()
    if cmd == "mask":
        mask()
    elif cmd == "unmask":
        unmask()
    else:
        print(f"Unknown command: {cmd}")

# 使用方法：
# 加密：python scripts/secrets_handler.py mask
# 解密：python scripts/secrets_handler.py unmask