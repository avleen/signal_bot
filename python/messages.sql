CREATE TABLE IF NOT EXISTS `messages` (
  `id` integer not null primary key autoincrement,
  `timestamp` UNSIGNED BIG INT null,
  `sourceNumber` TEXT null,
  `sourceName` TEXT not null,
  `message` TEXT not null,
  `groupId` TEXT not null,
  `mentions` TEXT,
  `created_at` datetime not null default CURRENT_TIMESTAMP);
