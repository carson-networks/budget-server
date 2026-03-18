CREATE TABLE budgets (
    category_id UUID NOT NULL,
    month       DATE NOT NULL,
    amount      DECIMAL(100, 4) NOT NULL,
    CONSTRAINT pk_budgets PRIMARY KEY (category_id, month),
    CONSTRAINT fk_budgets_category FOREIGN KEY (category_id) REFERENCES categories(id)
);

CREATE INDEX idx_budgets_category_month_desc ON budgets (category_id, month DESC);
