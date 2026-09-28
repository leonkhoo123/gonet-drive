import OtpInput from 'react-otp-input';

const otpInputClasses = [
  "w-10 h-12 sm:w-12 sm:h-14 text-center text-lg font-semibold",
  "border-2 rounded-xl bg-background transition-all duration-150",
  "focus:ring-2 focus:ring-primary/50 focus:border-primary focus:outline-none",
  "[&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none [-moz-appearance:textfield]",
].join(' ');

interface OtpCodeInputProps {
  value: string;
  onChange: (value: string) => void;
}

const OtpCodeInput: React.FC<OtpCodeInputProps> = ({ value, onChange }) => (
  <div className="flex justify-center w-full">
    <OtpInput
      value={value}
      onChange={onChange}
      numInputs={6}
      shouldAutoFocus
      renderSeparator={<span className="mx-0.5 sm:mx-1 text-muted-foreground/30 select-none">&bull;</span>}
      renderInput={(props) => (
        <input
          {...props}
          type="number"
          inputMode="numeric"
          pattern="[0-9]*"
          className={otpInputClasses}
          style={{ width: "2.5rem" }}
        />
      )}
    />
  </div>
);

export default OtpCodeInput;
