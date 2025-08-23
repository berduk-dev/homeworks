CREATE TABLE public.links (
    id bigserial primary key,
    long_link text NOT NULL,
    short_link text NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.redirects (
    id bigserial primary key,
    long_link text NOT NULL,
    short_link text NOT NULL,
    user_agent text NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL
);