create table Logs (
  INT_ID serial not null primary key,
  EXT_LOOKUP_KEY bytea not null unique,
  HEAD bytea
);

