"use client";

import * as React from "react";
import { Controller, type FieldValues, type FieldPath } from "react-hook-form";
import { ChevronDown } from "lucide-react";

import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
} from "@/components/ui/field";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

import type { FormInputProps } from "./type";

export const DEFAULT_CURRENCIES = [
  "IDR",
  "USD",
  "EUR",
  "SGD",
  "JPY",
  "GBP",
] as const;

function getCurrencySymbol(currency: string) {
  try {
    const part = new Intl.NumberFormat(undefined, {
      style: "currency",
      currency,
      currencyDisplay: "narrowSymbol",
    })
      .formatToParts(0)
      .find((piece) => piece.type === "currency");
    return part?.value ?? currency;
  } catch {
    return currency;
  }
}

function formatAmountForDisplay(raw: string) {
  if (!raw) return "";

  const [wholePart, decimalPart] = raw.split(".");
  const wholeFormatted = wholePart
    ? new Intl.NumberFormat(undefined).format(Number(wholePart) || 0)
    : "";

  if (decimalPart === undefined) {
    return raw.endsWith(".") ? `${wholeFormatted}.` : wholeFormatted;
  }
  return `${wholeFormatted}.${decimalPart}`;
}

function sanitizeAmountInput(input: string) {
  let value = input.replace(/[^0-9.]/g, "");

  const firstDot = value.indexOf(".");
  if (firstDot !== -1) {
    value =
      value.slice(0, firstDot + 1) +
      value.slice(firstDot + 1).replace(/\./g, "");
    const [whole, decimal] = value.split(".");
    value = `${whole}.${decimal.slice(0, 2)}`;
  }

  return value.replace(/^0+(?=\d)/, "");
}

type MoneyInputFieldProps<T extends FieldValues> = FormInputProps<T> & {
  /** Field holding the selected currency code. Renders a currency dropdown when set. */
  currencyName?: FieldPath<T>;
  /** Currency codes offered by the dropdown when `currencyName` is set. */
  currencies?: readonly string[];
  /** Currency to display when there's no dropdown (currency is fixed, e.g. the account's). */
  currency?: string;
  placeholder?: string;
  disabled?: boolean;
};

function MoneyInputField<T extends FieldValues>({
  control,
  name,
  currencyName,
  currencies = DEFAULT_CURRENCIES,
  currency: fixedCurrency = "IDR",
  label,
  description,
  className,
  placeholder,
  disabled,
}: MoneyInputFieldProps<T>) {
  return (
    <Controller
      control={control}
      name={name}
      render={({
        field: { value, onChange, ...amountField },
        fieldState: { invalid, isTouched, isDirty, error },
      }) => {
        const rawValue = (value as string | undefined) ?? "";

        return (
          <Field
            name={name}
            invalid={invalid}
            touched={isTouched}
            dirty={isDirty}
            className={className}
          >
            {label ? <FieldLabel>{label}</FieldLabel> : null}
            <InputGroup className="w-full">
              <InputGroupAddon align="inline-start">
                {currencyName ? (
                  <Controller
                    control={control}
                    name={currencyName}
                    render={({ field: currencyField }) => {
                      const selected =
                        (currencyField.value as string | undefined) ??
                        currencies[0];

                      return (
                        <DropdownMenu>
                          <DropdownMenuTrigger
                            render={
                              <InputGroupButton
                                type="button"
                                disabled={disabled}
                                className="gap-1 font-semibold text-foreground"
                              >
                                {selected}
                                <ChevronDown
                                  className="size-3.5 text-muted-foreground"
                                  aria-hidden
                                />
                              </InputGroupButton>
                            }
                          />
                          <DropdownMenuContent align="start">
                            <DropdownMenuRadioGroup
                              value={selected}
                              onValueChange={(next) =>
                                currencyField.onChange(next)
                              }
                            >
                              {currencies.map((code) => (
                                <DropdownMenuRadioItem
                                  key={code}
                                  value={code}
                                  closeOnClick
                                >
                                  <span className="font-medium">{code}</span>
                                  <span className="ml-1 text-muted-foreground">
                                    {getCurrencySymbol(code)}
                                  </span>
                                </DropdownMenuRadioItem>
                              ))}
                            </DropdownMenuRadioGroup>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      );
                    }}
                  />
                ) : (
                  <span className="font-semibold text-foreground">
                    {getCurrencySymbol(fixedCurrency)}
                  </span>
                )}
              </InputGroupAddon>
              <InputGroupInput
                id={name}
                inputMode="decimal"
                placeholder={placeholder}
                disabled={disabled}
                value={formatAmountForDisplay(rawValue)}
                onChange={(event) =>
                  onChange(sanitizeAmountInput(event.target.value))
                }
                {...amountField}
              />
            </InputGroup>
            {description ? (
              <FieldDescription>{description}</FieldDescription>
            ) : null}
            <FieldError match={!!error}>{error?.message}</FieldError>
          </Field>
        );
      }}
    />
  );
}

export { MoneyInputField };
