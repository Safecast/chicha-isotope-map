#!/bin/bash
# Fetch and populate usernames for existing uploads from Safecast API

# Connect to database and get distinct user IDs
PGPASSWORD= psql -h 127.0.0.1 -U postgres -d safecast -t -c "
SELECT DISTINCT user_id 
FROM uploads 
WHERE user_id IS NOT NULL 
  AND user_id != '' 
  AND username IS NULL
ORDER BY user_id;
" | while read -r user_id; do
    # Trim whitespace
    user_id=$(echo "$user_id" | xargs)
    
    if [ -z "$user_id" ]; then
        continue
    fi
    
    echo "Fetching username for user ID: $user_id"
    
    # Fetch user info from Safecast API
    response=$(curl -s "https://api.safecast.org/users/${user_id}.json")
    
    # Extract username using jq
    username=$(echo "$response" | jq -r '.name // empty')
    
    if [ -n "$username" ]; then
        echo "  Found: $username"
        # Update database
        PGPASSWORD= psql -h 127.0.0.1 -U postgres -d safecast -c "
UPDATE uploads 
SET username = '$username' 
WHERE user_id = '$user_id' AND username IS NULL;
" > /dev/null
    else
        echo "  Not found or error"
    fi
    
    # Rate limit to avoid overwhelming the API
    sleep 0.5
done

echo "Username population complete!"
