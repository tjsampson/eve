alter table job
    add success_exit_codes varchar(100) default '0' not null;

alter table service
    add success_exit_codes varchar(100) default '0' not null;