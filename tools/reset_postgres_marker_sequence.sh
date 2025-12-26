#!/bin/bash
# Reset the PostgreSQL sequence for markers table to avoid duplicate key errors

echo "Resetting markers sequence..."
sudo -u postgres psql -d safecast << 'EOF'
SELECT setval('markers_id_seq', (SELECT MAX(id) FROM markers) + 1);
EOF

echo "Sequence reset complete!"
