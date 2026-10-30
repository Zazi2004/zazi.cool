-- Habits, the days they apply to, and the days they were done.
-- Every date is a civil date (no time of day): the "day" is computed in Go with TZ_NAME.
CREATE TABLE habits (
    id          bigserial PRIMARY KEY,
    name        text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 60),
    created_on  date NOT NULL,
    -- First day the habit no longer applies; null while it is active.
    archived_on date
);

-- Two active habits cannot share a name; archived ones do not count.
CREATE UNIQUE INDEX habits_active_name_idx ON habits (name) WHERE archived_on IS NULL;

-- Versioned schedule: one row per change of weekdays, effective from valid_from on, so
-- editing a habit never rewrites how past days were counted.
-- weekdays is a 7-bit mask: bit 0 = Monday ... bit 6 = Sunday.
CREATE TABLE habit_schedules (
    habit_id   bigint   NOT NULL REFERENCES habits (id) ON DELETE CASCADE,
    valid_from date     NOT NULL,
    weekdays   smallint NOT NULL CHECK (weekdays BETWEEN 1 AND 127),
    PRIMARY KEY (habit_id, valid_from)
);

-- A row means "done"; unchecking deletes it.
CREATE TABLE completions (
    habit_id   bigint      NOT NULL REFERENCES habits (id) ON DELETE CASCADE,
    day        date        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (habit_id, day)
);
