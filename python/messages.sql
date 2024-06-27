CREATE TABLE `messages` (
  `id` integer not null primary key autoincrement,
  `timestamp` UNSIGNED BIG INT null,
  `sourceNumber` TEXT null,
  `sourceName` TEXT not null,
  `message` TEXT not null,
  `groupId` TEXT not null,
  `created_at` datetime not null default CURRENT_TIMESTAMP);
