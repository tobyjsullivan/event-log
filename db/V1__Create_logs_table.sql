create table Logs (
  INT_ID serial not null primary key,
  EXT_LOOKUP_KEY bit(128) not null unique,
  HEAD bit(256)
);

