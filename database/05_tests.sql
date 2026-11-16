INSERT INTO title_rating (profile_id, title_id, score)
VALUES (1, '9afdf956-9c75-d5d1-2f55-b20b44d54ba7', 15);

-- Result: [HY000][3819] Check constraint 'chk_title_rating_score' is violated.

INSERT INTO watchlist (profile_id, title_id) VALUES (1, '9afdf956-9c75-d5d1-2f55-b20b44d54ba7'); 

INSERT INTO watchlist (profile_id, title_id) VALUES (1, '9afdf956-9c75-d5d1-2f55-b20b44d54ba7'); 

-- Result: [23000][1062] Duplicate entry '1-9afdf956-9c75-d5d1-2f55-b20b44d54ba7' for key 'watchlist.PRIMARY'


-- to be continued...