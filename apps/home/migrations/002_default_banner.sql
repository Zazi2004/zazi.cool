-- The gallery of built-in illustrations arrives: a page that never had its content
-- saved (version 1) and shows no banner gets the first illustration (F8). Choosing
-- "none" afterwards raises the version, so it is never overridden.
UPDATE content SET banner_choice = 'aube' WHERE banner_choice = 'none' AND version = 1;
