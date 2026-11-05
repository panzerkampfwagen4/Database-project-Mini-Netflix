import pymysql
import sys
import time

def parse_sql_statements(sql):
    statements = []
    current_statement = []
    in_string = False
    escape = False
    
    i = 0
    n = len(sql)
    while i < n:
        c = sql[i]
        
        # Check single line comments --
        if not in_string and c == '-' and i + 1 < n and sql[i+1] == '-':
            # Skip until end of line
            while i < n and sql[i] != '\n':
                i += 1
            continue
            
        if in_string:
            if escape:
                escape = False
            elif c == '\\':
                escape = True
            elif c == "'":
                if i + 1 < n and sql[i+1] == "'": # doubled quote ''
                    current_statement.append(c)
                    current_statement.append(sql[i+1])
                    i += 2
                    continue
                else:
                    in_string = False
            current_statement.append(c)
        else:
            if c == "'":
                in_string = True
                current_statement.append(c)
            elif c == ';':
                stmt = ''.join(current_statement).strip()
                if stmt:
                    statements.append(stmt)
                current_statement = []
            else:
                current_statement.append(c)
        i += 1
        
    tail = ''.join(current_statement).strip()
    if tail:
        statements.append(tail)
        
    return statements

def main():
    print("Reading seed.sql...")
    with open('seed.sql', 'r', encoding='utf-8') as f:
        content = f.read()

    statements = parse_sql_statements(content)
    print(f"Parsed {len(statements)} SQL statements.")

    print("Connecting to TiDB Cloud (AWS Singapore)...")
    conn = pymysql.connect(
        host='gateway01.ap-southeast-1.prod.aws.tidbcloud.com',
        port=4000,
        user='2AmEQpexzMhSUci.root',
        password='TdsLpNo74ZKkqeSy',
        database='mininetflix',
        ssl={'ca': None},
        autocommit=False,
        charset='utf8mb4'
    )

    start_time = time.time()
    with conn.cursor() as cur:
        # Disable foreign key checks for bulk import
        cur.execute("SET FOREIGN_KEY_CHECKS = 0;")
        cur.execute("TRUNCATE TABLE payment_transaction;")
        cur.execute("TRUNCATE TABLE credit_card;")
        
        for i, stmt in enumerate(statements, 1):
            # Skip interior SET FOREIGN_KEY_CHECKS = 1 so FK remains disabled until all rows loaded
            if "SET FOREIGN_KEY_CHECKS" in stmt.upper() and "= 1" in stmt:
                continue
            try:
                cur.execute(stmt)
                if i % 50 == 0 or i == len(statements):
                    conn.commit()
                    pct = (i / len(statements)) * 100
                    print(f"[{pct:5.1f}%] Executed {i}/{len(statements)} statements...")
            except Exception as e:
                print(f"\n[ERROR at statement #{i}]: {e}\nStatement preview: {stmt[:160]}")
                conn.rollback()
                raise e

        # Re-enable foreign key checks and commit
        cur.execute("SET FOREIGN_KEY_CHECKS = 1;")
        conn.commit()

    total_time = time.time() - start_time
    print(f"\n>>> Successfully imported seed.sql in {total_time:.2f} seconds. <<<")

    # Verify counts
    print("\n--- Verifying Row Counts in TiDB Cloud ---")
    tables = [
        'catalog_title', 'movie_title', 'tv_series', 
        'season', 'episode', 'streamable_media_unit', 
        'subscription_tier', 'account', 'profile', 
        'watchlist', 'title_rating', 'watch_history',
        'credit_card', 'payment_transaction'
    ]
    with conn.cursor() as cur:
        for t in tables:
            cur.execute(f"SELECT COUNT(*) FROM `{t}`")
            cnt = cur.fetchone()[0]
            print(f"- {t:24}: {cnt:>6} rows")

    conn.close()

if __name__ == '__main__':
    main()
