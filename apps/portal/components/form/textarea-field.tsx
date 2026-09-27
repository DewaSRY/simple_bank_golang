"use client";

import * as React from "react";
import { Controller, type FieldValues } from "react-hook-form";
import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
} from "@/components/ui/field";
import { Textarea } from "@/components/ui/textarea";

import type { FormInputProps } from "./type";

type CounterPosition = "inside" | "top" | "bottom";

type TextareaFieldProps<T extends FieldValues> = FormInputProps<T> & {
  counter?: boolean;
  counterPosition?: CounterPosition;
} & Omit<
    React.ComponentProps<typeof Textarea>,
    "name" | "value" | "defaultValue" | "onChange" | "onBlur"
  >;

function TextareaField<T extends FieldValues>({
  control,
  name,
  label,
  description,
  className,
  counter = false,
  counterPosition = "bottom",
  maxLength,
  ...textareaProps
}: TextareaFieldProps<T>) {
  return (
    <Controller
      control={control}
      name={name}
      render={({
        field: { value, ...field },
        fieldState: { invalid, isTouched, isDirty, error },
      }) => {
        const length = (value ?? "").length;
        const counterText = maxLength ? `${length}/${maxLength}` : `${length}`;

        const bottomCounter = counter && counterPosition === "bottom";

        // A plain <textarea> isn't a Base UI Field.Control, so the label
        // association and invalid state have to be wired by hand here.
        return (
          <Field
            name={name}
            invalid={invalid}
            touched={isTouched}
            dirty={isDirty}
            className={className}
          >
            {label || (counter && counterPosition === "top") ? (
              <div className="flex w-full items-center justify-between">
                {label ? <FieldLabel htmlFor={name}>{label}</FieldLabel> : <span />}
                {counter && counterPosition === "top" ? (
                  <span className="text-xs text-muted-foreground tabular-nums">
                    {counterText}
                  </span>
                ) : null}
              </div>
            ) : null}
            <div className="relative w-full">
              <Textarea
                id={name}
                {...textareaProps}
                {...field}
                aria-invalid={invalid || undefined}
                value={value ?? ""}
                maxLength={maxLength}
                className={
                  counter && counterPosition === "inside" ? "pb-7" : undefined
                }
              />
              {counter && counterPosition === "inside" ? (
                <span className="pointer-events-none absolute right-2.5 bottom-2 text-xs text-muted-foreground tabular-nums">
                  {counterText}
                </span>
              ) : null}
            </div>
            {description || error || bottomCounter ? (
              <div className="flex w-full items-start justify-between gap-3">
                <div className="min-w-0">
                  <FieldError match={!!error}>{error?.message}</FieldError>
                  {description && !error ? (
                    <FieldDescription>{description}</FieldDescription>
                  ) : null}
                </div>
                {bottomCounter ? (
                  <span className="shrink-0 text-xs text-muted-foreground tabular-nums">
                    {counterText}
                  </span>
                ) : null}
              </div>
            ) : null}
          </Field>
        );
      }}
    />
  );
}

export { TextareaField };
