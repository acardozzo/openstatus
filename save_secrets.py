import os
import subprocess

# Read .env files recursively
def read_env_files(directory):
    env_data = {}
    for root, _, files in os.walk(directory):
        for file in files:
            if file == '.env':
                with open(os.path.join(root, file)) as f:
                    for line in f:
                        if '=' in line and not line.startswith('#'):
                            key, value = line.strip().split('=', 1)
                            env_data[key] = value
    return env_data

# Update or Create Secret using GitHub CLI (gh)
def update_secret(secret_name, secret_value):
    try:
        subprocess.run(
            [
                'gh', 'secret', 'set', secret_name,
                '--body', secret_value
            ],
            check=True
        )
        print(f'Updated secret: {secret_name}')
    except subprocess.CalledProcessError as e:
        print(f'Failed to update secret {secret_name}: {e}')

# Main function to run the script
def run():
    try:
        root_env = read_env_files(os.getcwd())
        for key, value in root_env.items():
            update_secret(key, value)
        print('All secrets updated successfully.')
    except Exception as e:
        print(f'Error updating secrets: {e}')

if __name__ == '__main__':
    run()
