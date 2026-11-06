-- Habits v2: the v1 habit tables are dropped (their data is abandoned, spec §1) and the
-- whole v2 schema is created. `sessions` is untouched.
-- Every date is a civil date (no time of day): the "day" is computed in Go with TZ_NAME.
DROP TABLE completions;
DROP TABLE habit_schedules;
DROP TABLE habits;

CREATE TABLE habits (
    id          bigserial PRIMARY KEY,
    name        text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 60),
    emoji       text NOT NULL CHECK (char_length(emoji) BETWEEN 1 AND 16),
    color       text NOT NULL CHECK (color IN ('amber', 'violet', 'blue', 'red', 'green', 'teal', 'slate', 'pink')),
    moment      text NOT NULL CHECK (moment IN ('morning', 'day', 'evening')),
    -- Manual order within the moment.
    position    int  NOT NULL,
    -- Empty for a yes/no habit.
    unit        text NOT NULL DEFAULT '' CHECK (char_length(unit) <= 20),
    -- Increment of one tap.
    step        int  NOT NULL DEFAULT 1 CHECK (step >= 1),
    created_on  date NOT NULL,
    -- First day the habit no longer applies; null while it is active.
    archived_on date
);

-- Two active habits cannot share a name; archived ones do not count.
CREATE UNIQUE INDEX habits_active_name_idx ON habits (name) WHERE archived_on IS NULL;

-- Versioned rule: one row per change of period, target or weekdays, effective from
-- valid_from on. A row for a past period is never rewritten, so editing a habit never
-- changes how the past was counted.
-- weekdays is a 7-bit mask: bit 0 = Monday ... bit 6 = Sunday.
CREATE TABLE habit_rules (
    habit_id   bigint   NOT NULL REFERENCES habits (id) ON DELETE CASCADE,
    valid_from date     NOT NULL,
    period     text     NOT NULL CHECK (period IN ('day', 'week', 'month', 'year')),
    target     int      NOT NULL CHECK (target BETWEEN 1 AND 10000),
    weekdays   smallint NOT NULL CHECK (weekdays BETWEEN 1 AND 127),
    PRIMARY KEY (habit_id, valid_from)
);

-- What was done on a day; a value of 0 deletes the row.
CREATE TABLE entries (
    habit_id   bigint      NOT NULL REFERENCES habits (id) ON DELETE CASCADE,
    day        date        NOT NULL,
    value      int         NOT NULL CHECK (value > 0),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (habit_id, day)
);

CREATE TABLE day_notes (
    day        date        PRIMARY KEY,
    text       text        NOT NULL CHECK (char_length(text) <= 1000),
    updated_at timestamptz NOT NULL DEFAULT now()
);
