#ifdef GODUCKDB_FROM_SOURCE
/*
Copyright 2018-2023 Stichting DuckDB Foundation

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

#pragma once
#define DUCKDB_AMALGAMATION 1
#define DUCKDB_SOURCE_ID "6536a77232"
#define DUCKDB_VERSION "v0.8.1"
//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/main/connection.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/profiler_format.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/constants.hpp
//
//
//===----------------------------------------------------------------------===//



#include <memory>
//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/string.hpp
//
//
//===----------------------------------------------------------------------===//



#include <string>
#include <sstream>

namespace duckdb {
using std::string;
}

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/main/winapi.hpp
//
//
//===----------------------------------------------------------------------===//



#ifndef DUCKDB_API
#if defined(_WIN32) && !defined(__MINGW32__)
#if defined(DUCKDB_BUILD_LIBRARY) && !defined(DUCKDB_BUILD_LOADABLE_EXTENSION)
#define DUCKDB_API __declspec(dllexport)
#else
#define DUCKDB_API __declspec(dllimport)
#endif
#else
#define DUCKDB_API
#endif
#endif

#ifndef DUCKDB_EXTENSION_API
#ifdef _WIN32
#ifdef DUCKDB_BUILD_LOADABLE_EXTENSION
#define DUCKDB_EXTENSION_API __declspec(dllexport)
#else
#define DUCKDB_EXTENSION_API
#endif
#else
#define DUCKDB_EXTENSION_API __attribute__((visibility("default")))
#endif
#endif



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/exception.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/assert.hpp
//
//
//===----------------------------------------------------------------------===//





#if (defined(DUCKDB_USE_STANDARD_ASSERT) || !defined(DEBUG)) && !defined(DUCKDB_FORCE_ASSERT)

#include <assert.h>
#define D_ASSERT assert
namespace duckdb {
DUCKDB_API void DuckDBAssertInternal(bool condition, const char *condition_name, const char *file, int linenr);
}

#else
namespace duckdb {
DUCKDB_API void DuckDBAssertInternal(bool condition, const char *condition_name, const char *file, int linenr);
}

#define D_ASSERT(condition) duckdb::DuckDBAssertInternal(bool(condition), #condition, __FILE__, __LINE__)

#endif

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/exception_format_value.hpp
//
//
//===----------------------------------------------------------------------===//








#include <stdint.h>

namespace duckdb {

struct hugeint_t {
public:
	uint64_t lower;
	int64_t upper;

public:
	hugeint_t() = default;
	DUCKDB_API hugeint_t(int64_t value); // NOLINT: Allow implicit conversion from `int64_t`
	constexpr hugeint_t(int64_t upper, uint64_t lower) : lower(lower), upper(upper) {
	}
	constexpr hugeint_t(const hugeint_t &rhs) = default;
	constexpr hugeint_t(hugeint_t &&rhs) = default;
	hugeint_t &operator=(const hugeint_t &rhs) = default;
	hugeint_t &operator=(hugeint_t &&rhs) = default;

	DUCKDB_API string ToString() const;

	// comparison operators
	DUCKDB_API bool operator==(const hugeint_t &rhs) const;
	DUCKDB_API bool operator!=(const hugeint_t &rhs) const;
	DUCKDB_API bool operator<=(const hugeint_t &rhs) const;
	DUCKDB_API bool operator<(const hugeint_t &rhs) const;
	DUCKDB_API bool operator>(const hugeint_t &rhs) const;
	DUCKDB_API bool operator>=(const hugeint_t &rhs) const;

	// arithmetic operators
	DUCKDB_API hugeint_t operator+(const hugeint_t &rhs) const;
	DUCKDB_API hugeint_t operator-(const hugeint_t &rhs) const;
	DUCKDB_API hugeint_t operator*(const hugeint_t &rhs) const;
	DUCKDB_API hugeint_t operator/(const hugeint_t &rhs) const;
	DUCKDB_API hugeint_t operator%(const hugeint_t &rhs) const;
	DUCKDB_API hugeint_t operator-() const;

	// bitwise operators
	DUCKDB_API hugeint_t operator>>(const hugeint_t &rhs) const;
	DUCKDB_API hugeint_t operator<<(const hugeint_t &rhs) const;
	DUCKDB_API hugeint_t operator&(const hugeint_t &rhs) const;
	DUCKDB_API hugeint_t operator|(const hugeint_t &rhs) const;
	DUCKDB_API hugeint_t operator^(const hugeint_t &rhs) const;
	DUCKDB_API hugeint_t operator~() const;

	// in-place operators
	DUCKDB_API hugeint_t &operator+=(const hugeint_t &rhs);
	DUCKDB_API hugeint_t &operator-=(const hugeint_t &rhs);
	DUCKDB_API hugeint_t &operator*=(const hugeint_t &rhs);
	DUCKDB_API hugeint_t &operator/=(const hugeint_t &rhs);
	DUCKDB_API hugeint_t &operator%=(const hugeint_t &rhs);
	DUCKDB_API hugeint_t &operator>>=(const hugeint_t &rhs);
	DUCKDB_API hugeint_t &operator<<=(const hugeint_t &rhs);
	DUCKDB_API hugeint_t &operator&=(const hugeint_t &rhs);
	DUCKDB_API hugeint_t &operator|=(const hugeint_t &rhs);
	DUCKDB_API hugeint_t &operator^=(const hugeint_t &rhs);
};

} // namespace duckdb


#include <vector>

namespace duckdb {

// Helper class to support custom overloading
// Escaping " and quoting the value with "
class SQLIdentifier {
public:
	SQLIdentifier(const string &raw_string) : raw_string(raw_string) {
	}

public:
	string raw_string;
};

// Helper class to support custom overloading
// Escaping ' and quoting the value with '
class SQLString {
public:
	SQLString(const string &raw_string) : raw_string(raw_string) {
	}

public:
	string raw_string;
};

enum class PhysicalType : uint8_t;
struct LogicalType;

enum class ExceptionFormatValueType : uint8_t {
	FORMAT_VALUE_TYPE_DOUBLE,
	FORMAT_VALUE_TYPE_INTEGER,
	FORMAT_VALUE_TYPE_STRING
};

struct ExceptionFormatValue {
	DUCKDB_API ExceptionFormatValue(double dbl_val);   // NOLINT
	DUCKDB_API ExceptionFormatValue(int64_t int_val);  // NOLINT
	DUCKDB_API ExceptionFormatValue(string str_val);   // NOLINT
	DUCKDB_API ExceptionFormatValue(hugeint_t hg_val); // NOLINT

	ExceptionFormatValueType type;

	double dbl_val = 0;
	int64_t int_val = 0;
	string str_val;

public:
	template <class T>
	static ExceptionFormatValue CreateFormatValue(T value) {
		return int64_t(value);
	}
	static string Format(const string &msg, std::vector<ExceptionFormatValue> &values);
};

template <>
DUCKDB_API ExceptionFormatValue ExceptionFormatValue::CreateFormatValue(PhysicalType value);
template <>
DUCKDB_API ExceptionFormatValue ExceptionFormatValue::CreateFormatValue(SQLString value);
template <>
DUCKDB_API ExceptionFormatValue ExceptionFormatValue::CreateFormatValue(SQLIdentifier value);
template <>
DUCKDB_API ExceptionFormatValue ExceptionFormatValue::CreateFormatValue(LogicalType value);
template <>
DUCKDB_API ExceptionFormatValue ExceptionFormatValue::CreateFormatValue(float value);
template <>
DUCKDB_API ExceptionFormatValue ExceptionFormatValue::CreateFormatValue(double value);
template <>
DUCKDB_API ExceptionFormatValue ExceptionFormatValue::CreateFormatValue(string value);
template <>
DUCKDB_API ExceptionFormatValue ExceptionFormatValue::CreateFormatValue(const char *value);
template <>
DUCKDB_API ExceptionFormatValue ExceptionFormatValue::CreateFormatValue(char *value);
template <>
DUCKDB_API ExceptionFormatValue ExceptionFormatValue::CreateFormatValue(hugeint_t value);

} // namespace duckdb



#include <memory>

namespace duckdb {

using std::make_shared;
using std::shared_ptr;
using std::weak_ptr;

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/map.hpp
//
//
//===----------------------------------------------------------------------===//



#include <map>

namespace duckdb {
using std::map;
using std::multimap;
} // namespace duckdb



#include <cstdint>

namespace duckdb {

//! a saner size_t for loop indices etc
typedef uint64_t idx_t;

//! The type used for row identifiers
typedef int64_t row_t;

//! The type used for hashes
typedef uint64_t hash_t;

//! data pointers
typedef uint8_t data_t;
typedef data_t *data_ptr_t;
typedef const data_t *const_data_ptr_t;

//! Type used for the selection vector
typedef uint32_t sel_t;
//! Type used for transaction timestamps
typedef idx_t transaction_t;

//! Type used for column identifiers
typedef idx_t column_t;
//! Type used for storage (column) identifiers
typedef idx_t storage_t;

template <class SRC>
data_ptr_t data_ptr_cast(SRC *src) {
	return reinterpret_cast<data_ptr_t>(src);
}

template <class SRC>
const_data_ptr_t const_data_ptr_cast(const SRC *src) {
	return reinterpret_cast<const_data_ptr_t>(src);
}

template <class SRC>
char *char_ptr_cast(SRC *src) {
	return reinterpret_cast<char *>(src);
}

template <class SRC>
const char *const_char_ptr_cast(const SRC *src) {
	return reinterpret_cast<const char *>(src);
}

template <class SRC>
const unsigned char *const_uchar_ptr_cast(const SRC *src) {
	return reinterpret_cast<const unsigned char *>(src);
}

template <class SRC>
uintptr_t CastPointerToValue(SRC *src) {
	return uintptr_t(src);
}

} // namespace duckdb


#include <vector>
#include <stdexcept>

namespace duckdb {
enum class PhysicalType : uint8_t;
struct LogicalType;
struct hugeint_t;

inline void assert_restrict_function(const void *left_start, const void *left_end, const void *right_start,
                                     const void *right_end, const char *fname, int linenr) {
	// assert that the two pointers do not overlap
#ifdef DEBUG
	if (!(left_end <= right_start || right_end <= left_start)) {
		printf("ASSERT RESTRICT FAILED: %s:%d\n", fname, linenr);
		D_ASSERT(0);
	}
#endif
}

#define ASSERT_RESTRICT(left_start, left_end, right_start, right_end)                                                  \
	assert_restrict_function(left_start, left_end, right_start, right_end, __FILE__, __LINE__)

//===--------------------------------------------------------------------===//
// Exception Types
//===--------------------------------------------------------------------===//

enum class ExceptionType {
	INVALID = 0,          // invalid type
	OUT_OF_RANGE = 1,     // value out of range error
	CONVERSION = 2,       // conversion/casting error
	UNKNOWN_TYPE = 3,     // unknown type
	DECIMAL = 4,          // decimal related
	MISMATCH_TYPE = 5,    // type mismatch
	DIVIDE_BY_ZERO = 6,   // divide by 0
	OBJECT_SIZE = 7,      // object size exceeded
	INVALID_TYPE = 8,     // incompatible for operation
	SERIALIZATION = 9,    // serialization
	TRANSACTION = 10,     // transaction management
	NOT_IMPLEMENTED = 11, // method not implemented
	EXPRESSION = 12,      // expression parsing
	CATALOG = 13,         // catalog related
	PARSER = 14,          // parser related
	PLANNER = 15,         // planner related
	SCHEDULER = 16,       // scheduler related
	EXECUTOR = 17,        // executor related
	CONSTRAINT = 18,      // constraint related
	INDEX = 19,           // index related
	STAT = 20,            // stat related
	CONNECTION = 21,      // connection related
	SYNTAX = 22,          // syntax related
	SETTINGS = 23,        // settings related
	BINDER = 24,          // binder related
	NETWORK = 25,         // network related
	OPTIMIZER = 26,       // optimizer related
	NULL_POINTER = 27,    // nullptr exception
	IO = 28,              // IO exception
	INTERRUPT = 29,       // interrupt
	FATAL = 30,           // Fatal exceptions are non-recoverable, and render the entire DB in an unusable state
	INTERNAL = 31,        // Internal exceptions indicate something went wrong internally (i.e. bug in the code base)
	INVALID_INPUT = 32,   // Input or arguments error
	OUT_OF_MEMORY = 33,   // out of memory
	PERMISSION = 34,      // insufficient permissions
	PARAMETER_NOT_RESOLVED = 35, // parameter types could not be resolved
	PARAMETER_NOT_ALLOWED = 36,  // parameter types not allowed
	DEPENDENCY = 37,             // dependency
	HTTP = 38,
	MISSING_EXTENSION = 39 // Thrown when an extension is used but not loaded
};
class HTTPException;

class Exception : public std::exception {
public:
	DUCKDB_API explicit Exception(const string &msg);
	DUCKDB_API Exception(ExceptionType exception_type, const string &message);

	ExceptionType type;

public:
	DUCKDB_API const char *what() const noexcept override;
	DUCKDB_API const string &RawMessage() const;

	DUCKDB_API static string ExceptionTypeToString(ExceptionType type);
	[[noreturn]] DUCKDB_API static void ThrowAsTypeWithMessage(ExceptionType type, const string &message,
	                                                           const std::shared_ptr<Exception> &original);
	virtual std::shared_ptr<Exception> Copy() const {
		return make_shared<Exception>(type, raw_message_);
	}
	DUCKDB_API const HTTPException &AsHTTPException() const;

	template <typename... Args>
	static string ConstructMessage(const string &msg, Args... params) {
		const std::size_t num_args = sizeof...(Args);
		if (num_args == 0)
			return msg;
		std::vector<ExceptionFormatValue> values;
		return ConstructMessageRecursive(msg, values, params...);
	}

	DUCKDB_API static string ConstructMessageRecursive(const string &msg, std::vector<ExceptionFormatValue> &values);

	template <class T, typename... Args>
	static string ConstructMessageRecursive(const string &msg, std::vector<ExceptionFormatValue> &values, T param,
	                                        Args... params) {
		values.push_back(ExceptionFormatValue::CreateFormatValue<T>(param));
		return ConstructMessageRecursive(msg, values, params...);
	}

	DUCKDB_API static bool UncaughtException();

	DUCKDB_API static string GetStackTrace(int max_depth = 120);
	static string FormatStackTrace(string message = "") {
		return (message + "\n" + GetStackTrace());
	}

private:
	string exception_message_;
	string raw_message_;
};

//===--------------------------------------------------------------------===//
// Exception derived classes
//===--------------------------------------------------------------------===//

//! Exceptions that are StandardExceptions do NOT invalidate the current transaction when thrown
class StandardException : public Exception {
public:
	DUCKDB_API StandardException(ExceptionType exception_type, const string &message);
};

class CatalogException : public StandardException {
public:
	DUCKDB_API explicit CatalogException(const string &msg);

	template <typename... Args>
	explicit CatalogException(const string &msg, Args... params) : CatalogException(ConstructMessage(msg, params...)) {
	}
};

class ConnectionException : public StandardException {
public:
	DUCKDB_API explicit ConnectionException(const string &msg);

	template <typename... Args>
	explicit ConnectionException(const string &msg, Args... params)
	    : ConnectionException(ConstructMessage(msg, params...)) {
	}
};

class ParserException : public StandardException {
public:
	DUCKDB_API explicit ParserException(const string &msg);

	template <typename... Args>
	explicit ParserException(const string &msg, Args... params) : ParserException(ConstructMessage(msg, params...)) {
	}
};

class PermissionException : public StandardException {
public:
	DUCKDB_API explicit PermissionException(const string &msg);

	template <typename... Args>
	explicit PermissionException(const string &msg, Args... params)
	    : PermissionException(ConstructMessage(msg, params...)) {
	}
};

class BinderException : public StandardException {
public:
	DUCKDB_API explicit BinderException(const string &msg);

	template <typename... Args>
	explicit BinderException(const string &msg, Args... params) : BinderException(ConstructMessage(msg, params...)) {
	}
};

class ConversionException : public Exception {
public:
	DUCKDB_API explicit ConversionException(const string &msg);

	template <typename... Args>
	explicit ConversionException(const string &msg, Args... params)
	    : ConversionException(ConstructMessage(msg, params...)) {
	}
};

class TransactionException : public Exception {
public:
	DUCKDB_API explicit TransactionException(const string &msg);

	template <typename... Args>
	explicit TransactionException(const string &msg, Args... params)
	    : TransactionException(ConstructMessage(msg, params...)) {
	}
};

class NotImplementedException : public Exception {
public:
	DUCKDB_API explicit NotImplementedException(const string &msg);

	template <typename... Args>
	explicit NotImplementedException(const string &msg, Args... params)
	    : NotImplementedException(ConstructMessage(msg, params...)) {
	}
};

class OutOfRangeException : public Exception {
public:
	DUCKDB_API explicit OutOfRangeException(const string &msg);

	template <typename... Args>
	explicit OutOfRangeException(const string &msg, Args... params)
	    : OutOfRangeException(ConstructMessage(msg, params...)) {
	}
};

class OutOfMemoryException : public Exception {
public:
	DUCKDB_API explicit OutOfMemoryException(const string &msg);

	template <typename... Args>
	explicit OutOfMemoryException(const string &msg, Args... params)
	    : OutOfMemoryException(ConstructMessage(msg, params...)) {
	}
};

class SyntaxException : public Exception {
public:
	DUCKDB_API explicit SyntaxException(const string &msg);

	template <typename... Args>
	explicit SyntaxException(const string &msg, Args... params) : SyntaxException(ConstructMessage(msg, params...)) {
	}
};

class ConstraintException : public Exception {
public:
	DUCKDB_API explicit ConstraintException(const string &msg);

	template <typename... Args>
	explicit ConstraintException(const string &msg, Args... params)
	    : ConstraintException(ConstructMessage(msg, params...)) {
	}
};

class DependencyException : public Exception {
public:
	DUCKDB_API explicit DependencyException(const string &msg);

	template <typename... Args>
	explicit DependencyException(const string &msg, Args... params)
	    : DependencyException(ConstructMessage(msg, params...)) {
	}
};

class IOException : public Exception {
public:
	DUCKDB_API explicit IOException(const string &msg);
	explicit IOException(ExceptionType exception_type, const string &msg) : Exception(exception_type, msg) {
	}

	template <typename... Args>
	explicit IOException(const string &msg, Args... params) : IOException(ConstructMessage(msg, params...)) {
	}
};

class MissingExtensionException : public Exception {
public:
	DUCKDB_API explicit MissingExtensionException(const string &msg);

	template <typename... Args>
	explicit MissingExtensionException(const string &msg, Args... params)
	    : MissingExtensionException(ConstructMessage(msg, params...)) {
	}
};

class HTTPException : public IOException {
public:
	template <typename>
	struct ResponseShape {
		typedef int status;
	};

	template <class RESPONSE, typename ResponseShape<decltype(RESPONSE::status)>::status = 0, typename... ARGS>
	explicit HTTPException(RESPONSE &response, const string &msg, ARGS... params)
	    : HTTPException(response.status, response.body, response.headers, response.reason, msg, params...) {
	}

	template <typename>
	struct ResponseWrapperShape {
		typedef int code;
	};
	template <class RESPONSE, typename ResponseWrapperShape<decltype(RESPONSE::code)>::code = 0, typename... ARGS>
	explicit HTTPException(RESPONSE &response, const string &msg, ARGS... params)
	    : HTTPException(response.code, response.body, response.headers, response.error, msg, params...) {
	}

	template <typename HEADERS, typename... ARGS>
	explicit HTTPException(int status_code, string response_body, HEADERS headers, const string &reason,
	                       const string &msg, ARGS... params)
	    : IOException(ExceptionType::HTTP, ConstructMessage(msg, params...)), status_code(status_code), reason(reason),
	      response_body(std::move(response_body)) {
		this->headers.insert(headers.begin(), headers.end());
		D_ASSERT(this->headers.size() > 0);
	}

	std::shared_ptr<Exception> Copy() const {
		return make_shared<HTTPException>(status_code, response_body, headers, reason, RawMessage());
	}

	const std::multimap<std::string, std::string> GetHeaders() const {
		return headers;
	}
	int GetStatusCode() const {
		return status_code;
	}
	const string &GetResponseBody() const {
		return response_body;
	}
	const string &GetReason() const {
		return reason;
	}
	[[noreturn]] void Throw() const {
		throw HTTPException(status_code, response_body, headers, reason, RawMessage());
	}

private:
	int status_code;
	string reason;
	string response_body;
	std::multimap<string, string> headers;
};

class SerializationException : public Exception {
public:
	DUCKDB_API explicit SerializationException(const string &msg);

	template <typename... Args>
	explicit SerializationException(const string &msg, Args... params)
	    : SerializationException(ConstructMessage(msg, params...)) {
	}
};

class SequenceException : public Exception {
public:
	DUCKDB_API explicit SequenceException(const string &msg);

	template <typename... Args>
	explicit SequenceException(const string &msg, Args... params)
	    : SequenceException(ConstructMessage(msg, params...)) {
	}
};

class InterruptException : public Exception {
public:
	DUCKDB_API InterruptException();
};

class FatalException : public Exception {
public:
	explicit FatalException(const string &msg) : FatalException(ExceptionType::FATAL, msg) {
	}
	template <typename... Args>
	explicit FatalException(const string &msg, Args... params) : FatalException(ConstructMessage(msg, params...)) {
	}

protected:
	DUCKDB_API explicit FatalException(ExceptionType type, const string &msg);
	template <typename... Args>
	explicit FatalException(ExceptionType type, const string &msg, Args... params)
	    : FatalException(type, ConstructMessage(msg, params...)) {
	}
};

class InternalException : public FatalException {
public:
	DUCKDB_API explicit InternalException(const string &msg);

	template <typename... Args>
	explicit InternalException(const string &msg, Args... params)
	    : InternalException(ConstructMessage(msg, params...)) {
	}
};

class InvalidInputException : public Exception {
public:
	DUCKDB_API explicit InvalidInputException(const string &msg);

	template <typename... Args>
	explicit InvalidInputException(const string &msg, Args... params)
	    : InvalidInputException(ConstructMessage(msg, params...)) {
	}
};

class CastException : public Exception {
public:
	DUCKDB_API CastException(const PhysicalType origType, const PhysicalType newType);
	DUCKDB_API CastException(const LogicalType &origType, const LogicalType &newType);
	DUCKDB_API
	CastException(const string &msg); //! Needed to be able to recreate the exception after it's been serialized
};

class InvalidTypeException : public Exception {
public:
	DUCKDB_API InvalidTypeException(PhysicalType type, const string &msg);
	DUCKDB_API InvalidTypeException(const LogicalType &type, const string &msg);
	DUCKDB_API
	InvalidTypeException(const string &msg); //! Needed to be able to recreate the exception after it's been serialized
};

class TypeMismatchException : public Exception {
public:
	DUCKDB_API TypeMismatchException(const PhysicalType type_1, const PhysicalType type_2, const string &msg);
	DUCKDB_API TypeMismatchException(const LogicalType &type_1, const LogicalType &type_2, const string &msg);
	DUCKDB_API
	TypeMismatchException(const string &msg); //! Needed to be able to recreate the exception after it's been serialized
};

class ValueOutOfRangeException : public Exception {
public:
	DUCKDB_API ValueOutOfRangeException(const int64_t value, const PhysicalType origType, const PhysicalType newType);
	DUCKDB_API ValueOutOfRangeException(const hugeint_t value, const PhysicalType origType, const PhysicalType newType);
	DUCKDB_API ValueOutOfRangeException(const double value, const PhysicalType origType, const PhysicalType newType);
	DUCKDB_API ValueOutOfRangeException(const PhysicalType varType, const idx_t length);
	DUCKDB_API ValueOutOfRangeException(
	    const string &msg); //! Needed to be able to recreate the exception after it's been serialized
};

class ParameterNotAllowedException : public StandardException {
public:
	DUCKDB_API explicit ParameterNotAllowedException(const string &msg);

	template <typename... Args>
	explicit ParameterNotAllowedException(const string &msg, Args... params)
	    : ParameterNotAllowedException(ConstructMessage(msg, params...)) {
	}
};

//! Special exception that should be thrown in the binder if parameter types could not be resolved
//! This will cause prepared statements to be forcibly rebound with the actual parameter values
//! This exception is fatal if thrown outside of the binder (i.e. it should never be thrown outside of the binder)
class ParameterNotResolvedException : public Exception {
public:
	DUCKDB_API explicit ParameterNotResolvedException();
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/likely.hpp
//
//
//===----------------------------------------------------------------------===//



#if __GNUC__
#define DUCKDB_BUILTIN_EXPECT(cond, expected_value) (__builtin_expect(cond, expected_value))
#else
#define DUCKDB_BUILTIN_EXPECT(cond, expected_value) (cond)
#endif

#define DUCKDB_LIKELY(...)   DUCKDB_BUILTIN_EXPECT((__VA_ARGS__), 1)
#define DUCKDB_UNLIKELY(...) DUCKDB_BUILTIN_EXPECT((__VA_ARGS__), 0)



namespace duckdb {

template <bool ENABLED>
struct MemorySafety {
#ifdef DEBUG
	// In DEBUG mode safety is always on
	static constexpr bool enabled = true;
#else
	static constexpr bool enabled = ENABLED;
#endif
};

} // namespace duckdb


#include <memory>
#include <type_traits>

namespace duckdb {

template <class _Tp, class _Dp = std::default_delete<_Tp>, bool SAFE = true>
class unique_ptr : public std::unique_ptr<_Tp, _Dp> {
public:
	using original = std::unique_ptr<_Tp, _Dp>;
	using original::original;

private:
	static inline void AssertNotNull(const bool null) {
#if defined(DUCKDB_DEBUG_NO_SAFETY) || defined(DUCKDB_CLANG_TIDY)
		return;
#else
		if (DUCKDB_UNLIKELY(null)) {
			throw duckdb::InternalException("Attempted to dereference unique_ptr that is NULL!");
		}
#endif
	}

public:
	typename std::add_lvalue_reference<_Tp>::type operator*() const {
		const auto ptr = original::get();
		if (MemorySafety<SAFE>::enabled) {
			AssertNotNull(!ptr);
		}
		return *ptr;
	}

	typename original::pointer operator->() const {
		const auto ptr = original::get();
		if (MemorySafety<SAFE>::enabled) {
			AssertNotNull(!ptr);
		}
		return ptr;
	}

#ifdef DUCKDB_CLANG_TIDY
	// This is necessary to tell clang-tidy that it reinitializes the variable after a move
	[[clang::reinitializes]]
#endif
	inline void
	reset(typename original::pointer ptr = typename original::pointer()) noexcept {
		original::reset(ptr);
	}
};

template <class _Tp, class _Dp, bool SAFE>
class unique_ptr<_Tp[], _Dp, SAFE> : public std::unique_ptr<_Tp[], std::default_delete<_Tp[]>> {
public:
	using original = std::unique_ptr<_Tp[], std::default_delete<_Tp[]>>;
	using original::original;

private:
	static inline void AssertNotNull(const bool null) {
#if defined(DUCKDB_DEBUG_NO_SAFETY) || defined(DUCKDB_CLANG_TIDY)
		return;
#else
		if (DUCKDB_UNLIKELY(null)) {
			throw duckdb::InternalException("Attempted to dereference unique_ptr that is NULL!");
		}
#endif
	}

public:
	typename std::add_lvalue_reference<_Tp>::type operator[](size_t __i) const {
		const auto ptr = original::get();
		if (MemorySafety<SAFE>::enabled) {
			AssertNotNull(!ptr);
		}
		return ptr[__i];
	}
};

template <typename T>
using unique_array = unique_ptr<T[], std::default_delete<T>, true>;

template <typename T>
using unsafe_unique_array = unique_ptr<T[], std::default_delete<T>, false>;

template <typename T>
using unsafe_unique_ptr = unique_ptr<T, std::default_delete<T>, false>;

} // namespace duckdb



namespace duckdb {
class Serializer;
class Deserializer;

//! inline std directives that we use frequently
#ifndef DUCKDB_DEBUG_MOVE
using std::move;
#endif

// NOTE: there is a copy of this in the Postgres' parser grammar (gram.y)
#define DEFAULT_SCHEMA  "main"
#define INVALID_SCHEMA  ""
#define INVALID_CATALOG ""
#define SYSTEM_CATALOG  "system"
#define TEMP_CATALOG    "temp"

DUCKDB_API bool IsInvalidSchema(const string &str);
DUCKDB_API bool IsInvalidCatalog(const string &str);

//! Special value used to signify the ROW ID of a table
DUCKDB_API extern const column_t COLUMN_IDENTIFIER_ROW_ID;
DUCKDB_API bool IsRowIdColumnId(column_t column_id);

//! The maximum row identifier used in tables
extern const row_t MAX_ROW_ID;

extern const transaction_t TRANSACTION_ID_START;
extern const transaction_t MAX_TRANSACTION_ID;
extern const transaction_t MAXIMUM_QUERY_ID;
extern const transaction_t NOT_DELETED_ID;

extern const double PI;

struct DConstants {
	//! The value used to signify an invalid index entry
	static constexpr const idx_t INVALID_INDEX = idx_t(-1);
};

struct Storage {
	//! The size of a hard disk sector, only really needed for Direct IO
	constexpr static int SECTOR_SIZE = 4096;
	//! Block header size for blocks written to the storage
	constexpr static int BLOCK_HEADER_SIZE = sizeof(uint64_t);
	// Size of a memory slot managed by the StorageManager. This is the quantum of allocation for Blocks on DuckDB. We
	// default to 256KB. (1 << 18)
	constexpr static int BLOCK_ALLOC_SIZE = 262144;
	//! The actual memory space that is available within the blocks
	constexpr static int BLOCK_SIZE = BLOCK_ALLOC_SIZE - BLOCK_HEADER_SIZE;
	//! The size of the headers. This should be small and written more or less atomically by the hard disk. We default
	//! to the page size, which is 4KB. (1 << 12)
	constexpr static int FILE_HEADER_SIZE = 4096;
};

struct LogicalIndex {
	explicit LogicalIndex(idx_t index) : index(index) {
	}

	idx_t index;

	inline bool operator==(const LogicalIndex &rhs) const {
		return index == rhs.index;
	};
	inline bool operator!=(const LogicalIndex &rhs) const {
		return index != rhs.index;
	};
	inline bool operator<(const LogicalIndex &rhs) const {
		return index < rhs.index;
	};
	bool IsValid() {
		return index != DConstants::INVALID_INDEX;
	}
};

struct PhysicalIndex {
	explicit PhysicalIndex(idx_t index) : index(index) {
	}

	idx_t index;

	inline bool operator==(const PhysicalIndex &rhs) const {
		return index == rhs.index;
	};
	inline bool operator!=(const PhysicalIndex &rhs) const {
		return index != rhs.index;
	};
	inline bool operator<(const PhysicalIndex &rhs) const {
		return index < rhs.index;
	};
	bool IsValid() {
		return index != DConstants::INVALID_INDEX;
	}
};

DUCKDB_API bool IsPowerOfTwo(uint64_t v);
DUCKDB_API uint64_t NextPowerOfTwo(uint64_t v);
DUCKDB_API uint64_t PreviousPowerOfTwo(uint64_t v);

} // namespace duckdb


namespace duckdb {

enum class ProfilerPrintFormat : uint8_t { QUERY_TREE, JSON, QUERY_TREE_OPTIMIZER };

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/serializer/buffered_file_writer.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/serializer.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/catalog/catalog.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/catalog/catalog_entry.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/common.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/helper.hpp
//
//
//===----------------------------------------------------------------------===//





#include <string.h>
#include <type_traits>

#ifdef _MSC_VER
#define suint64_t int64_t
#endif

#if defined(_WIN32) || defined(_WIN64)
#define DUCKDB_WINDOWS
#elif defined(__unix__) || defined(__unix) || (defined(__APPLE__) && defined(__MACH__))
#define DUCKDB_POSIX
#endif

namespace duckdb {

// explicit fallthrough for switch_statementss
#ifndef __has_cpp_attribute // For backwards compatibility
#define __has_cpp_attribute(x) 0
#endif
#if __has_cpp_attribute(clang::fallthrough)
#define DUCKDB_EXPLICIT_FALLTHROUGH [[clang::fallthrough]]
#elif __has_cpp_attribute(gnu::fallthrough)
#define DUCKDB_EXPLICIT_FALLTHROUGH [[gnu::fallthrough]]
#else
#define DUCKDB_EXPLICIT_FALLTHROUGH
#endif

template<class _Tp, bool SAFE = true>
struct __unique_if
{
    typedef unique_ptr<_Tp, std::default_delete<_Tp>, SAFE> __unique_single;
};

template<class _Tp>
struct __unique_if<_Tp[]>
{
    typedef unique_ptr<_Tp[]> __unique_array_unknown_bound;
};

template<class _Tp, size_t _Np>
struct __unique_if<_Tp[_Np]>
{
    typedef void __unique_array_known_bound;
};

template<class _Tp, class... _Args>
inline 
typename __unique_if<_Tp, true>::__unique_single
make_uniq(_Args&&... __args)
{
    return unique_ptr<_Tp, std::default_delete<_Tp>, true>(new _Tp(std::forward<_Args>(__args)...));
}

template<class _Tp, class... _Args>
inline 
typename __unique_if<_Tp, false>::__unique_single
make_unsafe_uniq(_Args&&... __args)
{
    return unique_ptr<_Tp, std::default_delete<_Tp>, false>(new _Tp(std::forward<_Args>(__args)...));
}

template<class _Tp>
inline unique_ptr<_Tp[], std::default_delete<_Tp>, true>
make_uniq_array(size_t __n)
{
    return unique_ptr<_Tp[], std::default_delete<_Tp>, true>(new _Tp[__n]());
}

template<class _Tp>
inline unique_ptr<_Tp[], std::default_delete<_Tp>, false>
make_unsafe_uniq_array(size_t __n)
{
    return unique_ptr<_Tp[], std::default_delete<_Tp>, false>(new _Tp[__n]());
}

template<class _Tp, class... _Args>
    typename __unique_if<_Tp>::__unique_array_known_bound
    make_uniq(_Args&&...) = delete;


template <typename S, typename T, typename... Args>
unique_ptr<S> make_uniq_base(Args &&... args) {
	return unique_ptr<S>(new T(std::forward<Args>(args)...));
}

#ifdef DUCKDB_ENABLE_DEPRECATED_API
template <typename S, typename T, typename... Args>
unique_ptr<S> make_unique_base(Args &&... args) {
	return unique_ptr<S>(new T(std::forward<Args>(args)...));
}
#endif // DUCKDB_ENABLE_DEPRECATED_API

template <typename T, typename S>
unique_ptr<S> unique_ptr_cast(unique_ptr<T> src) {
	return unique_ptr<S>(static_cast<S *>(src.release()));
}

struct SharedConstructor {
	template <class T, typename... ARGS>
	static shared_ptr<T> Create(ARGS &&...args) {
		return make_shared<T>(std::forward<ARGS>(args)...);
	}
};

struct UniqueConstructor {
	template <class T, typename... ARGS>
	static unique_ptr<T> Create(ARGS &&...args) {
		return make_uniq<T>(std::forward<ARGS>(args)...);
	}
};

#ifdef DUCKDB_DEBUG_MOVE
template<class T>
typename std::remove_reference<T>::type&& move(T&& t) noexcept {
	// the nonsensical sizeof check ensures this is never instantiated
	static_assert(sizeof(T) == 0, "Use std::move instead of unqualified move or duckdb::move");
}
#endif

template <class T, class... _Args>
static duckdb::unique_ptr<T> make_unique(_Args&&... __args) {
#ifndef DUCKDB_ENABLE_DEPRECATED_API
	static_assert(sizeof(T) == 0, "Use make_uniq instead of make_unique!");
#endif // DUCKDB_ENABLE_DEPRECATED_API
	return unique_ptr<T>(new T(std::forward<_Args>(__args)...));
}

template <typename T>
T MaxValue(T a, T b) {
	return a > b ? a : b;
}

template <typename T>
T MinValue(T a, T b) {
	return a < b ? a : b;
}

template <typename T>
T AbsValue(T a) {
	return a < 0 ? -a : a;
}

//Align value (ceiling)
template<class T, T val=8>
static inline T AlignValue(T n) {
	return ((n + (val - 1)) / val) * val;
}

template<class T, T val=8>
static inline bool ValueIsAligned(T n) {
	return (n % val) == 0;
}

template <typename T>
T SignValue(T a) {
	return a < 0 ? -1 : 1;
}

template <typename T>
const T Load(const_data_ptr_t ptr) {
	T ret;
	memcpy(&ret, ptr, sizeof(ret));
	return ret;
}

template <typename T>
void Store(const T &val, data_ptr_t ptr) {
	memcpy(ptr, (void *)&val, sizeof(val));
}

//! This assigns a shared pointer, but ONLY assigns if "target" is not equal to "source"
//! If this is often the case, this manner of assignment is significantly faster (~20X faster)
//! Since it avoids the need of an atomic incref/decref at the cost of a single pointer comparison
//! Benchmark: https://gist.github.com/Mytherin/4db3faa8e233c4a9b874b21f62bb4b96
//! If the shared pointers are not the same, the penalty is very low (on the order of 1%~ slower)
//! This method should always be preferred if there is a (reasonable) chance that the pointers are the same
template<class T>
void AssignSharedPointer(shared_ptr<T> &target, const shared_ptr<T> &source) {
	if (target.get() != source.get()) {
		target = source;
	}
}

template<typename T>
using reference = std::reference_wrapper<T>;

template<typename T>
using const_reference = std::reference_wrapper<const T>;

//! Returns whether or not two reference wrappers refer to the same object
template<class T>
bool RefersToSameObject(const reference<T> &A, const reference<T> &B) {
	return &A.get() == &B.get();
}

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/vector.hpp
//
//
//===----------------------------------------------------------------------===//








#include <vector>

namespace duckdb {

template <class _Tp, bool SAFE = true>
class vector : public std::vector<_Tp, std::allocator<_Tp>> {
public:
	using original = std::vector<_Tp, std::allocator<_Tp>>;
	using original::original;
	using size_type = typename original::size_type;
	using const_reference = typename original::const_reference;
	using reference = typename original::reference;

private:
	static inline void AssertIndexInBounds(idx_t index, idx_t size) {
#if defined(DUCKDB_DEBUG_NO_SAFETY) || defined(DUCKDB_CLANG_TIDY)
		return;
#else
		if (DUCKDB_UNLIKELY(index >= size)) {
			throw InternalException("Attempted to access index %ld within vector of size %ld", index, size);
		}
#endif
	}

public:
#ifdef DUCKDB_CLANG_TIDY
	// This is necessary to tell clang-tidy that it reinitializes the variable after a move
	[[clang::reinitializes]]
#endif
	inline void
	clear() noexcept {
		original::clear();
	}

	// Because we create the other constructor, the implicitly created constructor
	// gets deleted, so we have to be explicit
	vector() = default;
	vector(original &&other) : original(std::move(other)) {
	}
	template <bool _SAFE>
	vector(vector<_Tp, _SAFE> &&other) : original(std::move(other)) {
	}

	template <bool _SAFE = false>
	inline typename original::reference get(typename original::size_type __n) {
		if (MemorySafety<_SAFE>::enabled) {
			AssertIndexInBounds(__n, original::size());
		}
		return original::operator[](__n);
	}

	template <bool _SAFE = false>
	inline typename original::const_reference get(typename original::size_type __n) const {
		if (MemorySafety<_SAFE>::enabled) {
			AssertIndexInBounds(__n, original::size());
		}
		return original::operator[](__n);
	}

	typename original::reference operator[](typename original::size_type __n) {
		return get<SAFE>(__n);
	}
	typename original::const_reference operator[](typename original::size_type __n) const {
		return get<SAFE>(__n);
	}

	typename original::reference front() {
		return get<SAFE>(0);
	}

	typename original::const_reference front() const {
		return get<SAFE>(0);
	}

	typename original::reference back() {
		if (original::empty()) {
			throw InternalException("'back' called on an empty vector!");
		}
		return get<SAFE>(original::size() - 1);
	}

	typename original::const_reference back() const {
		if (original::empty()) {
			throw InternalException("'back' called on an empty vector!");
		}
		return get<SAFE>(original::size() - 1);
	}
};

template <typename T>
using unsafe_vector = vector<T, false>;

} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/catalog_type.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

//===--------------------------------------------------------------------===//
// Catalog Types
//===--------------------------------------------------------------------===//
enum class CatalogType : uint8_t {
	INVALID = 0,
	TABLE_ENTRY = 1,
	SCHEMA_ENTRY = 2,
	VIEW_ENTRY = 3,
	INDEX_ENTRY = 4,
	PREPARED_STATEMENT = 5,
	SEQUENCE_ENTRY = 6,
	COLLATION_ENTRY = 7,
	TYPE_ENTRY = 8,
	DATABASE_ENTRY = 9,

	// functions
	TABLE_FUNCTION_ENTRY = 25,
	SCALAR_FUNCTION_ENTRY = 26,
	AGGREGATE_FUNCTION_ENTRY = 27,
	PRAGMA_FUNCTION_ENTRY = 28,
	COPY_FUNCTION_ENTRY = 29,
	MACRO_ENTRY = 30,
	TABLE_MACRO_ENTRY = 31,

	// version info
	UPDATED_ENTRY = 50,
	DELETED_ENTRY = 51,
};

DUCKDB_API string CatalogTypeToString(CatalogType type);

} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/atomic.hpp
//
//
//===----------------------------------------------------------------------===//



#include <atomic>

namespace duckdb {
using std::atomic;
}

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/optional_ptr.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {

template <class T>
class optional_ptr {
public:
	optional_ptr() : ptr(nullptr) {
	}
	optional_ptr(T *ptr_p) : ptr(ptr_p) { // NOLINT: allow implicit creation from pointer
	}
	optional_ptr(const unique_ptr<T> &ptr_p) : ptr(ptr_p.get()) { // NOLINT: allow implicit creation from unique pointer
	}

	void CheckValid() const {
		if (!ptr) {
			throw InternalException("Attempting to dereference an optional pointer that is not set");
		}
	}

	operator bool() const {
		return ptr;
	}
	T &operator*() {
		CheckValid();
		return *ptr;
	}
	const T &operator*() const {
		CheckValid();
		return *ptr;
	}
	T *operator->() {
		CheckValid();
		return ptr;
	}
	const T *operator->() const {
		CheckValid();
		return ptr;
	}
	T *get() {
		// CheckValid();
		return ptr;
	}
	const T *get() const {
		// CheckValid();
		return ptr;
	}
	// this looks dirty - but this is the default behavior of raw pointers
	T *get_mutable() const {
		// CheckValid();
		return ptr;
	}

	bool operator==(const optional_ptr<T> &rhs) const {
		return ptr == rhs.ptr;
	}

	bool operator!=(const optional_ptr<T> &rhs) const {
		return ptr != rhs.ptr;
	}

private:
	T *ptr;
};

} // namespace duckdb

#include <memory>

namespace duckdb {
struct AlterInfo;
class Catalog;
class CatalogSet;
class ClientContext;
class SchemaCatalogEntry;

//! Abstract base class of an entry in the catalog
class CatalogEntry {
public:
	CatalogEntry(CatalogType type, Catalog &catalog, string name);
	CatalogEntry(CatalogType type, string name, idx_t oid);
	virtual ~CatalogEntry();

	//! The oid of the entry
	idx_t oid;
	//! The type of this catalog entry
	CatalogType type;
	//! Reference to the catalog set this entry is stored in
	optional_ptr<CatalogSet> set;
	//! The name of the entry
	string name;
	//! Whether or not the object is deleted
	bool deleted;
	//! Whether or not the object is temporary and should not be added to the WAL
	bool temporary;
	//! Whether or not the entry is an internal entry (cannot be deleted, not dumped, etc)
	bool internal;
	//! Timestamp at which the catalog entry was created
	atomic<transaction_t> timestamp;
	//! Child entry
	unique_ptr<CatalogEntry> child;
	//! Parent entry (the node that dependents_map this node)
	optional_ptr<CatalogEntry> parent;

public:
	virtual unique_ptr<CatalogEntry> AlterEntry(ClientContext &context, AlterInfo &info);
	virtual void UndoAlter(ClientContext &context, AlterInfo &info);

	virtual unique_ptr<CatalogEntry> Copy(ClientContext &context) const;

	//! Sets the CatalogEntry as the new root entry (i.e. the newest entry)
	// this is called on a rollback to an AlterEntry
	virtual void SetAsRoot();

	//! Convert the catalog entry to a SQL string that can be used to re-construct the catalog entry
	virtual string ToSQL() const;

	virtual Catalog &ParentCatalog();
	virtual SchemaCatalogEntry &ParentSchema();

	virtual void Verify(Catalog &catalog);

public:
	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}
	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
};

class InCatalogEntry : public CatalogEntry {
public:
	InCatalogEntry(CatalogType type, Catalog &catalog, string name);
	~InCatalogEntry() override;

	//! The catalog the entry belongs to
	Catalog &catalog;

public:
	Catalog &ParentCatalog() override {
		return catalog;
	}

	void Verify(Catalog &catalog) override;
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/mutex.hpp
//
//
//===----------------------------------------------------------------------===//



#include <mutex>

namespace duckdb {
using std::lock_guard;
using std::mutex;
using std::unique_lock;
} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/query_error_context.hpp
//
//
//===----------------------------------------------------------------------===//








namespace duckdb {
class SQLStatement;

class QueryErrorContext {
public:
	explicit QueryErrorContext(optional_ptr<SQLStatement> statement_ = nullptr,
	                           idx_t query_location_ = DConstants::INVALID_INDEX)
	    : statement(statement_), query_location(query_location_) {
	}

	//! The query statement
	optional_ptr<SQLStatement> statement;
	//! The location in which the error should be thrown
	idx_t query_location;

public:
	DUCKDB_API static string Format(const string &query, const string &error_message, int error_location);

	DUCKDB_API string FormatErrorRecursive(const string &msg, vector<ExceptionFormatValue> &values);
	template <class T, typename... Args>
	string FormatErrorRecursive(const string &msg, vector<ExceptionFormatValue> &values, T param, Args... params) {
		values.push_back(ExceptionFormatValue::CreateFormatValue<T>(param));
		return FormatErrorRecursive(msg, values, params...);
	}

	template <typename... Args>
	string FormatError(const string &msg, Args... params) {
		vector<ExceptionFormatValue> values;
		return FormatErrorRecursive(msg, values, params...);
	}
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/catalog/catalog_transaction.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {
class Catalog;
class ClientContext;
class DatabaseInstance;
class Transaction;

struct CatalogTransaction {
	CatalogTransaction(Catalog &catalog, ClientContext &context);
	CatalogTransaction(DatabaseInstance &db, transaction_t transaction_id_p, transaction_t start_time_p);

	optional_ptr<DatabaseInstance> db;
	optional_ptr<ClientContext> context;
	optional_ptr<Transaction> transaction;
	transaction_t transaction_id;
	transaction_t start_time;

	ClientContext &GetContext();

	static CatalogTransaction GetSystemTransaction(DatabaseInstance &db);
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/reference_map.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/unordered_map.hpp
//
//
//===----------------------------------------------------------------------===//



#include <unordered_map>

namespace duckdb {
using std::unordered_map;
}

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/unordered_set.hpp
//
//
//===----------------------------------------------------------------------===//



#include <unordered_set>

namespace duckdb {
using std::unordered_set;
}


namespace duckdb {
class Expression;

template <class T>
struct ReferenceHashFunction {
	uint64_t operator()(const reference<T> &ref) const {
		return std::hash<void *>()((void *)&ref.get());
	}
};

template <class T>
struct ReferenceEquality {
	bool operator()(const reference<T> &a, const reference<T> &b) const {
		return &a.get() == &b.get();
	}
};

template <typename T, typename TGT>
using reference_map_t = unordered_map<reference<T>, TGT, ReferenceHashFunction<T>, ReferenceEquality<T>>;

template <typename T>
using reference_set_t = unordered_set<reference<T>, ReferenceHashFunction<T>, ReferenceEquality<T>>;

} // namespace duckdb



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/on_entry_not_found.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

enum class OnEntryNotFound : uint8_t { THROW_EXCEPTION = 0, RETURN_NULL = 1 };

} // namespace duckdb

#include <functional>

namespace duckdb {
struct CreateSchemaInfo;
struct DropInfo;
struct BoundCreateTableInfo;
struct AlterTableInfo;
struct CreateTableFunctionInfo;
struct CreateCopyFunctionInfo;
struct CreatePragmaFunctionInfo;
struct CreateFunctionInfo;
struct CreateViewInfo;
struct CreateSequenceInfo;
struct CreateCollationInfo;
struct CreateIndexInfo;
struct CreateTypeInfo;
struct CreateTableInfo;
struct DatabaseSize;

class AttachedDatabase;
class ClientContext;
class Transaction;

class AggregateFunctionCatalogEntry;
class CollateCatalogEntry;
class SchemaCatalogEntry;
class TableCatalogEntry;
class ViewCatalogEntry;
class SequenceCatalogEntry;
class TableFunctionCatalogEntry;
class CopyFunctionCatalogEntry;
class PragmaFunctionCatalogEntry;
class CatalogSet;
class DatabaseInstance;
class DependencyManager;

struct CatalogLookup;
struct CatalogEntryLookup;
struct SimilarCatalogEntry;

class Binder;
class LogicalOperator;
class PhysicalOperator;
class LogicalCreateIndex;
class LogicalCreateTable;
class LogicalInsert;
class LogicalDelete;
class LogicalUpdate;
class CreateStatement;

//! The Catalog object represents the catalog of the database.
class Catalog {
public:
	explicit Catalog(AttachedDatabase &db);
	virtual ~Catalog();

public:
	//! Get the SystemCatalog from the ClientContext
	DUCKDB_API static Catalog &GetSystemCatalog(ClientContext &context);
	//! Get the SystemCatalog from the DatabaseInstance
	DUCKDB_API static Catalog &GetSystemCatalog(DatabaseInstance &db);
	//! Get the specified Catalog from the ClientContext
	DUCKDB_API static Catalog &GetCatalog(ClientContext &context, const string &catalog_name);
	//! Get the specified Catalog from the DatabaseInstance
	DUCKDB_API static Catalog &GetCatalog(DatabaseInstance &db, const string &catalog_name);
	//! Gets the specified Catalog from the database if it exists
	DUCKDB_API static optional_ptr<Catalog> GetCatalogEntry(ClientContext &context, const string &catalog_name);
	//! Get the specific Catalog from the AttachedDatabase
	DUCKDB_API static Catalog &GetCatalog(AttachedDatabase &db);

	DUCKDB_API AttachedDatabase &GetAttached();
	DUCKDB_API DatabaseInstance &GetDatabase();

	virtual bool IsDuckCatalog() {
		return false;
	}
	virtual void Initialize(bool load_builtin) = 0;

	bool IsSystemCatalog() const;
	bool IsTemporaryCatalog() const;

	//! Returns the current version of the catalog (incremented whenever anything changes, not stored between restarts)
	DUCKDB_API idx_t GetCatalogVersion();
	//! Trigger a modification in the catalog, increasing the catalog version and returning the previous version
	DUCKDB_API idx_t ModifyCatalog();

	//! Returns the catalog name - based on how the catalog was attached
	DUCKDB_API const string &GetName();
	DUCKDB_API idx_t GetOid();
	DUCKDB_API virtual string GetCatalogType() = 0;

	DUCKDB_API CatalogTransaction GetCatalogTransaction(ClientContext &context);

	//! Creates a schema in the catalog.
	DUCKDB_API virtual optional_ptr<CatalogEntry> CreateSchema(CatalogTransaction transaction,
	                                                           CreateSchemaInfo &info) = 0;
	DUCKDB_API optional_ptr<CatalogEntry> CreateSchema(ClientContext &context, CreateSchemaInfo &info);
	//! Creates a table in the catalog.
	DUCKDB_API optional_ptr<CatalogEntry> CreateTable(CatalogTransaction transaction, BoundCreateTableInfo &info);
	DUCKDB_API optional_ptr<CatalogEntry> CreateTable(ClientContext &context, BoundCreateTableInfo &info);
	//! Creates a table in the catalog.
	DUCKDB_API optional_ptr<CatalogEntry> CreateTable(ClientContext &context, unique_ptr<CreateTableInfo> info);
	//! Create a table function in the catalog
	DUCKDB_API optional_ptr<CatalogEntry> CreateTableFunction(CatalogTransaction transaction,
	                                                          CreateTableFunctionInfo &info);
	DUCKDB_API optional_ptr<CatalogEntry> CreateTableFunction(ClientContext &context, CreateTableFunctionInfo &info);
	// Kept for backwards compatibility
	DUCKDB_API optional_ptr<CatalogEntry> CreateTableFunction(ClientContext &context,
	                                                          optional_ptr<CreateTableFunctionInfo> info);
	//! Create a copy function in the catalog
	DUCKDB_API optional_ptr<CatalogEntry> CreateCopyFunction(CatalogTransaction transaction,
	                                                         CreateCopyFunctionInfo &info);
	DUCKDB_API optional_ptr<CatalogEntry> CreateCopyFunction(ClientContext &context, CreateCopyFunctionInfo &info);
	//! Create a pragma function in the catalog
	DUCKDB_API optional_ptr<CatalogEntry> CreatePragmaFunction(CatalogTransaction transaction,
	                                                           CreatePragmaFunctionInfo &info);
	DUCKDB_API optional_ptr<CatalogEntry> CreatePragmaFunction(ClientContext &context, CreatePragmaFunctionInfo &info);
	//! Create a scalar or aggregate function in the catalog
	DUCKDB_API optional_ptr<CatalogEntry> CreateFunction(CatalogTransaction transaction, CreateFunctionInfo &info);
	DUCKDB_API optional_ptr<CatalogEntry> CreateFunction(ClientContext &context, CreateFunctionInfo &info);
	//! Creates a table in the catalog.
	DUCKDB_API optional_ptr<CatalogEntry> CreateView(CatalogTransaction transaction, CreateViewInfo &info);
	DUCKDB_API optional_ptr<CatalogEntry> CreateView(ClientContext &context, CreateViewInfo &info);
	//! Creates a sequence in the catalog.
	DUCKDB_API optional_ptr<CatalogEntry> CreateSequence(CatalogTransaction transaction, CreateSequenceInfo &info);
	DUCKDB_API optional_ptr<CatalogEntry> CreateSequence(ClientContext &context, CreateSequenceInfo &info);
	//! Creates a Enum in the catalog.
	DUCKDB_API optional_ptr<CatalogEntry> CreateType(CatalogTransaction transaction, CreateTypeInfo &info);
	DUCKDB_API optional_ptr<CatalogEntry> CreateType(ClientContext &context, CreateTypeInfo &info);
	//! Creates a collation in the catalog
	DUCKDB_API optional_ptr<CatalogEntry> CreateCollation(CatalogTransaction transaction, CreateCollationInfo &info);
	DUCKDB_API optional_ptr<CatalogEntry> CreateCollation(ClientContext &context, CreateCollationInfo &info);
	//! Creates an index in the catalog
	DUCKDB_API optional_ptr<CatalogEntry> CreateIndex(CatalogTransaction transaction, CreateIndexInfo &info);
	DUCKDB_API optional_ptr<CatalogEntry> CreateIndex(ClientContext &context, CreateIndexInfo &info);

	//! Creates a table in the catalog.
	DUCKDB_API optional_ptr<CatalogEntry> CreateTable(CatalogTransaction transaction, SchemaCatalogEntry &schema,
	                                                  BoundCreateTableInfo &info);
	//! Create a table function in the catalog
	DUCKDB_API optional_ptr<CatalogEntry>
	CreateTableFunction(CatalogTransaction transaction, SchemaCatalogEntry &schema, CreateTableFunctionInfo &info);
	//! Create a copy function in the catalog
	DUCKDB_API optional_ptr<CatalogEntry> CreateCopyFunction(CatalogTransaction transaction, SchemaCatalogEntry &schema,
	                                                         CreateCopyFunctionInfo &info);
	//! Create a pragma function in the catalog
	DUCKDB_API optional_ptr<CatalogEntry>
	CreatePragmaFunction(CatalogTransaction transaction, SchemaCatalogEntry &schema, CreatePragmaFunctionInfo &info);
	//! Create a scalar or aggregate function in the catalog
	DUCKDB_API optional_ptr<CatalogEntry> CreateFunction(CatalogTransaction transaction, SchemaCatalogEntry &schema,
	                                                     CreateFunctionInfo &info);
	//! Creates a view in the catalog
	DUCKDB_API optional_ptr<CatalogEntry> CreateView(CatalogTransaction transaction, SchemaCatalogEntry &schema,
	                                                 CreateViewInfo &info);
	//! Creates a table in the catalog.
	DUCKDB_API optional_ptr<CatalogEntry> CreateSequence(CatalogTransaction transaction, SchemaCatalogEntry &schema,
	                                                     CreateSequenceInfo &info);
	//! Creates a enum in the catalog.
	DUCKDB_API optional_ptr<CatalogEntry> CreateType(CatalogTransaction transaction, SchemaCatalogEntry &schema,
	                                                 CreateTypeInfo &info);
	//! Creates a collation in the catalog
	DUCKDB_API optional_ptr<CatalogEntry> CreateCollation(CatalogTransaction transaction, SchemaCatalogEntry &schema,
	                                                      CreateCollationInfo &info);

	//! Drops an entry from the catalog
	DUCKDB_API void DropEntry(ClientContext &context, DropInfo &info);

	//! Returns the schema object with the specified name, or throws an exception if it does not exist
	DUCKDB_API SchemaCatalogEntry &GetSchema(ClientContext &context, const string &name,
	                                         QueryErrorContext error_context = QueryErrorContext());
	DUCKDB_API optional_ptr<SchemaCatalogEntry> GetSchema(ClientContext &context, const string &name,
	                                                      OnEntryNotFound if_not_found,
	                                                      QueryErrorContext error_context = QueryErrorContext());
	DUCKDB_API SchemaCatalogEntry &GetSchema(CatalogTransaction transaction, const string &name,
	                                         QueryErrorContext error_context = QueryErrorContext());
	DUCKDB_API virtual optional_ptr<SchemaCatalogEntry>
	GetSchema(CatalogTransaction transaction, const string &schema_name, OnEntryNotFound if_not_found,
	          QueryErrorContext error_context = QueryErrorContext()) = 0;
	DUCKDB_API static SchemaCatalogEntry &GetSchema(ClientContext &context, const string &catalog_name,
	                                                const string &schema_name,
	                                                QueryErrorContext error_context = QueryErrorContext());
	DUCKDB_API static optional_ptr<SchemaCatalogEntry> GetSchema(ClientContext &context, const string &catalog_name,
	                                                             const string &schema_name,
	                                                             OnEntryNotFound if_not_found,
	                                                             QueryErrorContext error_context = QueryErrorContext());
	//! Scans all the schemas in the system one-by-one, invoking the callback for each entry
	DUCKDB_API virtual void ScanSchemas(ClientContext &context, std::function<void(SchemaCatalogEntry &)> callback) = 0;
	//! Gets the "schema.name" entry of the specified type, if entry does not exist behavior depends on OnEntryNotFound
	DUCKDB_API optional_ptr<CatalogEntry> GetEntry(ClientContext &context, CatalogType type, const string &schema,
	                                               const string &name, OnEntryNotFound if_not_found,
	                                               QueryErrorContext error_context = QueryErrorContext());
	DUCKDB_API CatalogEntry &GetEntry(ClientContext &context, CatalogType type, const string &schema,
	                                  const string &name, QueryErrorContext error_context = QueryErrorContext());
	//! Gets the "catalog.schema.name" entry of the specified type, if entry does not exist behavior depends on
	//! OnEntryNotFound
	DUCKDB_API static optional_ptr<CatalogEntry> GetEntry(ClientContext &context, CatalogType type,
	                                                      const string &catalog, const string &schema,
	                                                      const string &name, OnEntryNotFound if_not_found,
	                                                      QueryErrorContext error_context = QueryErrorContext());
	DUCKDB_API static CatalogEntry &GetEntry(ClientContext &context, CatalogType type, const string &catalog,
	                                         const string &schema, const string &name,
	                                         QueryErrorContext error_context = QueryErrorContext());

	//! Gets the "schema.name" entry without a specified type, if entry does not exist an exception is thrown
	DUCKDB_API CatalogEntry &GetEntry(ClientContext &context, const string &schema, const string &name);

	//! Fetches a logical type from the catalog
	DUCKDB_API LogicalType GetType(ClientContext &context, const string &schema, const string &names,
	                               OnEntryNotFound if_not_found);

	DUCKDB_API static LogicalType GetType(ClientContext &context, const string &catalog_name, const string &schema,
	                                      const string &name);

	template <class T>
	optional_ptr<T> GetEntry(ClientContext &context, const string &schema_name, const string &name,
	                         OnEntryNotFound if_not_found, QueryErrorContext error_context = QueryErrorContext()) {
		auto entry = GetEntry(context, T::Type, schema_name, name, if_not_found, error_context);
		if (!entry) {
			return nullptr;
		}
		if (entry->type != T::Type) {
			throw CatalogException(error_context.FormatError("%s is not an %s", name, T::Name));
		}
		return &entry->template Cast<T>();
	}
	template <class T>
	T &GetEntry(ClientContext &context, const string &schema_name, const string &name,
	            QueryErrorContext error_context = QueryErrorContext()) {
		auto entry = GetEntry<T>(context, schema_name, name, OnEntryNotFound::THROW_EXCEPTION, error_context);
		return *entry;
	}

	//! Append a scalar or aggregate function to the catalog
	DUCKDB_API optional_ptr<CatalogEntry> AddFunction(ClientContext &context, CreateFunctionInfo &info);

	//! Alter an existing entry in the catalog.
	DUCKDB_API void Alter(ClientContext &context, AlterInfo &info);

	virtual unique_ptr<PhysicalOperator> PlanCreateTableAs(ClientContext &context, LogicalCreateTable &op,
	                                                       unique_ptr<PhysicalOperator> plan) = 0;
	virtual unique_ptr<PhysicalOperator> PlanInsert(ClientContext &context, LogicalInsert &op,
	                                                unique_ptr<PhysicalOperator> plan) = 0;
	virtual unique_ptr<PhysicalOperator> PlanDelete(ClientContext &context, LogicalDelete &op,
	                                                unique_ptr<PhysicalOperator> plan) = 0;
	virtual unique_ptr<PhysicalOperator> PlanUpdate(ClientContext &context, LogicalUpdate &op,
	                                                unique_ptr<PhysicalOperator> plan) = 0;
	virtual unique_ptr<LogicalOperator> BindCreateIndex(Binder &binder, CreateStatement &stmt, TableCatalogEntry &table,
	                                                    unique_ptr<LogicalOperator> plan) = 0;

	virtual DatabaseSize GetDatabaseSize(ClientContext &context) = 0;

	virtual bool InMemory() = 0;
	virtual string GetDBPath() = 0;

public:
	template <class T>
	static optional_ptr<T> GetEntry(ClientContext &context, const string &catalog_name, const string &schema_name,
	                                const string &name, OnEntryNotFound if_not_found,
	                                QueryErrorContext error_context = QueryErrorContext()) {
		auto entry = GetEntry(context, T::Type, catalog_name, schema_name, name, if_not_found, error_context);
		if (!entry) {
			return nullptr;
		}
		if (entry->type != T::Type) {
			throw CatalogException(error_context.FormatError("%s is not an %s", name, T::Name));
		}
		return &entry->template Cast<T>();
	}
	template <class T>
	static T &GetEntry(ClientContext &context, const string &catalog_name, const string &schema_name,
	                   const string &name, QueryErrorContext error_context = QueryErrorContext()) {
		auto entry =
		    GetEntry<T>(context, catalog_name, schema_name, name, OnEntryNotFound::THROW_EXCEPTION, error_context);
		return *entry;
	}

	DUCKDB_API vector<reference<SchemaCatalogEntry>> GetSchemas(ClientContext &context);
	DUCKDB_API static vector<reference<SchemaCatalogEntry>> GetSchemas(ClientContext &context,
	                                                                   const string &catalog_name);
	DUCKDB_API static vector<reference<SchemaCatalogEntry>> GetAllSchemas(ClientContext &context);

	virtual void Verify();

	static CatalogException UnrecognizedConfigurationError(ClientContext &context, const string &name);

protected:
	//! Reference to the database
	AttachedDatabase &db;

private:
	CatalogEntryLookup LookupEntryInternal(CatalogTransaction transaction, CatalogType type, const string &schema,
	                                       const string &name);
	CatalogEntryLookup LookupEntry(ClientContext &context, CatalogType type, const string &schema, const string &name,
	                               OnEntryNotFound if_not_found, QueryErrorContext error_context = QueryErrorContext());
	static CatalogEntryLookup LookupEntry(ClientContext &context, vector<CatalogLookup> &lookups, CatalogType type,
	                                      const string &name, OnEntryNotFound if_not_found,
	                                      QueryErrorContext error_context = QueryErrorContext());

	//! Return an exception with did-you-mean suggestion.
	static CatalogException CreateMissingEntryException(ClientContext &context, const string &entry_name,
	                                                    CatalogType type,
	                                                    const reference_set_t<SchemaCatalogEntry> &schemas,
	                                                    QueryErrorContext error_context);

	//! Return the close entry name, the distance and the belonging schema.
	static SimilarCatalogEntry SimilarEntryInSchemas(ClientContext &context, const string &entry_name, CatalogType type,
	                                                 const reference_set_t<SchemaCatalogEntry> &schemas);

	virtual void DropSchema(ClientContext &context, DropInfo &info) = 0;

public:
	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}

	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
};

} // namespace duckdb




#include <type_traits>

namespace duckdb {

//! The Serialize class is a base class that can be used to serializing objects into a binary buffer
class Serializer {
private:
	uint64_t version = 0L;

public:
	bool is_query_plan = false;

	virtual ~Serializer() {
	}

	//! Sets the version of the serialization that writers are expected to use
	//! The version is mostly the most recent one, unless modifying old data or streaming to
	//! an older version
	void SetVersion(uint64_t v) {
		D_ASSERT(this->version == 0); // version can only be set once
		this->version = v;
	}

	//! Returns the version of serialization that writers are expected to use
	uint64_t GetVersion() {
		return version;
	}

	virtual void WriteData(const_data_ptr_t buffer, idx_t write_size) = 0;

	template <class T>
	void Write(T element) {
		static_assert(std::is_trivially_destructible<T>(), "Write element must be trivially destructible");

		WriteData(const_data_ptr_cast(&element), sizeof(T));
	}

	//! Write data from a string buffer directly (without length prefix)
	void WriteBufferData(const string &str) {
		WriteData(const_data_ptr_cast(str.c_str()), str.size());
	}
	//! Write a string with a length prefix
	void WriteString(const string &val) {
		WriteStringLen(const_data_ptr_cast(val.c_str()), val.size());
	}
	void WriteStringLen(const_data_ptr_t val, idx_t len) {
		Write<uint32_t>((uint32_t)len);
		if (len > 0) {
			WriteData(val, len);
		}
	}

	template <class T>
	void WriteList(const vector<unique_ptr<T>> &list) {
		Write<uint32_t>((uint32_t)list.size());
		for (auto &child : list) {
			child->Serialize(*this);
		}
	}

	void WriteStringVector(const vector<string> &list) {
		Write<uint32_t>((uint32_t)list.size());
		for (auto &child : list) {
			WriteString(child);
		}
	}

	template <class T>
	void WriteOptional(const unique_ptr<T> &element) {
		Write<bool>(element ? true : false);
		if (element) {
			element->Serialize(*this);
		}
	}
};

//! The Deserializer class assists in deserializing a binary blob back into an
//! object
class Deserializer {
private:
	uint64_t version = 0L;

public:
	virtual ~Deserializer() {
	}

	//! Sets the version of the serialization that readers are expected to use
	//! The version is mostly the most recent one, unless reading old data or streaming from
	//! an older version
	void SetVersion(uint64_t v) {
		D_ASSERT(this->version == 0); // version can only be set once
		this->version = v;
	}

	//! Returns the version of serialization that readers are expected to use
	uint64_t GetVersion() {
		return version;
	}

	//! Reads [read_size] bytes into the buffer
	virtual void ReadData(data_ptr_t buffer, idx_t read_size) = 0;

	//! Gets the context for the deserializer
	virtual ClientContext &GetContext() {
		throw InternalException("This deserializer does not have a client-context");
	};

	//! Gets the catalog for the deserializer
	virtual optional_ptr<Catalog> GetCatalog() {
		return nullptr;
	};

	template <class T>
	T Read() {
		T value;
		ReadData(data_ptr_cast(&value), sizeof(T));
		return value;
	}

	template <class T, typename... ARGS>
	void ReadList(vector<unique_ptr<T>> &list, ARGS &&... args) {
		auto select_count = Read<uint32_t>();
		for (uint32_t i = 0; i < select_count; i++) {
			auto child = T::Deserialize(*this, std::forward<ARGS>(args)...);
			list.push_back(std::move(child));
		}
	}

	template <class T, class RETURN_TYPE = T, typename... ARGS>
	unique_ptr<RETURN_TYPE> ReadOptional(ARGS &&... args) {
		auto has_entry = Read<bool>();
		if (has_entry) {
			return T::Deserialize(*this, std::forward<ARGS>(args)...);
		}
		return nullptr;
	}

	void ReadStringVector(vector<string> &list);
};

template <>
DUCKDB_API string Deserializer::Read();

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/file_system.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/file_compression_type.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

enum class FileCompressionType : uint8_t { AUTO_DETECT = 0, UNCOMPRESSED = 1, GZIP = 2, ZSTD = 3 };

FileCompressionType FileCompressionTypeFromString(const string &input);

} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/file_buffer.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/debug_initialize.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

enum class DebugInitialize : uint8_t { NO_INITIALIZE = 0, DEBUG_ZERO_INITIALIZE = 1, DEBUG_ONE_INITIALIZE = 2 };

} // namespace duckdb


namespace duckdb {
class Allocator;
struct FileHandle;

enum class FileBufferType : uint8_t { BLOCK = 1, MANAGED_BUFFER = 2, TINY_BUFFER = 3 };

//! The FileBuffer represents a buffer that can be read or written to a Direct IO FileHandle.
class FileBuffer {
public:
	//! Allocates a buffer of the specified size, with room for additional header bytes
	//! (typically 8 bytes). On return, this->AllocSize() >= this->size >= user_size.
	//! Our allocation size will always be page-aligned, which is necessary to support
	//! DIRECT_IO
	FileBuffer(Allocator &allocator, FileBufferType type, uint64_t user_size);
	FileBuffer(FileBuffer &source, FileBufferType type);

	virtual ~FileBuffer();

	Allocator &allocator;
	//! The type of the buffer
	FileBufferType type;
	//! The buffer that users can write to
	data_ptr_t buffer;
	//! The size of the portion that users can write to, this is equivalent to internal_size - BLOCK_HEADER_SIZE
	uint64_t size;

public:
	//! Read into the FileBuffer from the specified location.
	void Read(FileHandle &handle, uint64_t location);
	//! Write the contents of the FileBuffer to the specified location.
	void Write(FileHandle &handle, uint64_t location);

	void Clear();

	// Same rules as the constructor. We will add room for a header, in additio to
	// the requested user bytes. We will then sector-align the result.
	void Resize(uint64_t user_size);

	uint64_t AllocSize() const {
		return internal_size;
	}
	data_ptr_t InternalBuffer() {
		return internal_buffer;
	}

	struct MemoryRequirement {
		idx_t alloc_size;
		idx_t header_size;
	};

	MemoryRequirement CalculateMemory(uint64_t user_size);

	void Initialize(DebugInitialize info);

protected:
	//! The pointer to the internal buffer that will be read or written, including the buffer header
	data_ptr_t internal_buffer;
	//! The aligned size as passed to the constructor. This is the size that is read or written to disk.
	uint64_t internal_size;

	void ReallocBuffer(size_t malloc_size);
	void Init();
};

} // namespace duckdb



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/file_glob_options.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

enum class FileGlobOptions : uint8_t {
	DISALLOW_EMPTY = 0,
	ALLOW_EMPTY = 1,
};

} // namespace duckdb


#include <functional>

#undef CreateDirectory
#undef MoveFile
#undef RemoveDirectory

namespace duckdb {
class AttachedDatabase;
class ClientContext;
class DatabaseInstance;
class FileOpener;
class FileSystem;

enum class FileType {
	//! Regular file
	FILE_TYPE_REGULAR,
	//! Directory
	FILE_TYPE_DIR,
	//! FIFO named pipe
	FILE_TYPE_FIFO,
	//! Socket
	FILE_TYPE_SOCKET,
	//! Symbolic link
	FILE_TYPE_LINK,
	//! Block device
	FILE_TYPE_BLOCKDEV,
	//! Character device
	FILE_TYPE_CHARDEV,
	//! Unknown or invalid file handle
	FILE_TYPE_INVALID,
};

struct FileHandle {
public:
	DUCKDB_API FileHandle(FileSystem &file_system, string path);
	FileHandle(const FileHandle &) = delete;
	DUCKDB_API virtual ~FileHandle();

	DUCKDB_API int64_t Read(void *buffer, idx_t nr_bytes);
	DUCKDB_API int64_t Write(void *buffer, idx_t nr_bytes);
	DUCKDB_API void Read(void *buffer, idx_t nr_bytes, idx_t location);
	DUCKDB_API void Write(void *buffer, idx_t nr_bytes, idx_t location);
	DUCKDB_API void Seek(idx_t location);
	DUCKDB_API void Reset();
	DUCKDB_API idx_t SeekPosition();
	DUCKDB_API void Sync();
	DUCKDB_API void Truncate(int64_t new_size);
	DUCKDB_API string ReadLine();

	DUCKDB_API bool CanSeek();
	DUCKDB_API bool OnDiskFile();
	DUCKDB_API idx_t GetFileSize();
	DUCKDB_API FileType GetType();

	//! Closes the file handle.
	DUCKDB_API virtual void Close() = 0;

	string GetPath() const {
		return path;
	}

	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}
	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}

public:
	FileSystem &file_system;
	string path;
};

enum class FileLockType : uint8_t { NO_LOCK = 0, READ_LOCK = 1, WRITE_LOCK = 2 };

class FileFlags {
public:
	//! Open file with read access
	static constexpr uint8_t FILE_FLAGS_READ = 1 << 0;
	//! Open file with write access
	static constexpr uint8_t FILE_FLAGS_WRITE = 1 << 1;
	//! Use direct IO when reading/writing to the file
	static constexpr uint8_t FILE_FLAGS_DIRECT_IO = 1 << 2;
	//! Create file if not exists, can only be used together with WRITE
	static constexpr uint8_t FILE_FLAGS_FILE_CREATE = 1 << 3;
	//! Always create a new file. If a file exists, the file is truncated. Cannot be used together with CREATE.
	static constexpr uint8_t FILE_FLAGS_FILE_CREATE_NEW = 1 << 4;
	//! Open file in append mode
	static constexpr uint8_t FILE_FLAGS_APPEND = 1 << 5;
};

class FileSystem {
public:
	DUCKDB_API virtual ~FileSystem();

public:
	DUCKDB_API static constexpr FileLockType DEFAULT_LOCK = FileLockType::NO_LOCK;
	DUCKDB_API static constexpr FileCompressionType DEFAULT_COMPRESSION = FileCompressionType::UNCOMPRESSED;
	DUCKDB_API static FileSystem &GetFileSystem(ClientContext &context);
	DUCKDB_API static FileSystem &GetFileSystem(DatabaseInstance &db);
	DUCKDB_API static FileSystem &Get(AttachedDatabase &db);

	DUCKDB_API virtual unique_ptr<FileHandle> OpenFile(const string &path, uint8_t flags,
	                                                   FileLockType lock = DEFAULT_LOCK,
	                                                   FileCompressionType compression = DEFAULT_COMPRESSION,
	                                                   FileOpener *opener = nullptr);

	//! Read exactly nr_bytes from the specified location in the file. Fails if nr_bytes could not be read. This is
	//! equivalent to calling SetFilePointer(location) followed by calling Read().
	DUCKDB_API virtual void Read(FileHandle &handle, void *buffer, int64_t nr_bytes, idx_t location);
	//! Write exactly nr_bytes to the specified location in the file. Fails if nr_bytes could not be written. This is
	//! equivalent to calling SetFilePointer(location) followed by calling Write().
	DUCKDB_API virtual void Write(FileHandle &handle, void *buffer, int64_t nr_bytes, idx_t location);
	//! Read nr_bytes from the specified file into the buffer, moving the file pointer forward by nr_bytes. Returns the
	//! amount of bytes read.
	DUCKDB_API virtual int64_t Read(FileHandle &handle, void *buffer, int64_t nr_bytes);
	//! Write nr_bytes from the buffer into the file, moving the file pointer forward by nr_bytes.
	DUCKDB_API virtual int64_t Write(FileHandle &handle, void *buffer, int64_t nr_bytes);

	//! Returns the file size of a file handle, returns -1 on error
	DUCKDB_API virtual int64_t GetFileSize(FileHandle &handle);
	//! Returns the file last modified time of a file handle, returns timespec with zero on all attributes on error
	DUCKDB_API virtual time_t GetLastModifiedTime(FileHandle &handle);
	//! Returns the file last modified time of a file handle, returns timespec with zero on all attributes on error
	DUCKDB_API virtual FileType GetFileType(FileHandle &handle);
	//! Truncate a file to a maximum size of new_size, new_size should be smaller than or equal to the current size of
	//! the file
	DUCKDB_API virtual void Truncate(FileHandle &handle, int64_t new_size);

	//! Check if a directory exists
	DUCKDB_API virtual bool DirectoryExists(const string &directory);
	//! Create a directory if it does not exist
	DUCKDB_API virtual void CreateDirectory(const string &directory);
	//! Recursively remove a directory and all files in it
	DUCKDB_API virtual void RemoveDirectory(const string &directory);
	//! List files in a directory, invoking the callback method for each one with (filename, is_dir)
	DUCKDB_API virtual bool ListFiles(const string &directory,
	                                  const std::function<void(const string &, bool)> &callback,
	                                  FileOpener *opener = nullptr);

	//! Move a file from source path to the target, StorageManager relies on this being an atomic action for ACID
	//! properties
	DUCKDB_API virtual void MoveFile(const string &source, const string &target);
	//! Check if a file exists
	DUCKDB_API virtual bool FileExists(const string &filename);
	//! Check if path is pipe
	DUCKDB_API virtual bool IsPipe(const string &filename);
	//! Remove a file from disk
	DUCKDB_API virtual void RemoveFile(const string &filename);
	//! Sync a file handle to disk
	DUCKDB_API virtual void FileSync(FileHandle &handle);
	//! Sets the working directory
	DUCKDB_API static void SetWorkingDirectory(const string &path);
	//! Gets the working directory
	DUCKDB_API static string GetWorkingDirectory();
	//! Gets the users home directory
	DUCKDB_API static string GetHomeDirectory(optional_ptr<FileOpener> opener);
	//! Gets the users home directory
	DUCKDB_API virtual string GetHomeDirectory();
	//! Expands a given path, including e.g. expanding the home directory of the user
	DUCKDB_API static string ExpandPath(const string &path, optional_ptr<FileOpener> opener);
	//! Expands a given path, including e.g. expanding the home directory of the user
	DUCKDB_API virtual string ExpandPath(const string &path);
	//! Returns the system-available memory in bytes. Returns DConstants::INVALID_INDEX if the system function fails.
	DUCKDB_API static idx_t GetAvailableMemory();
	//! Path separator for the current file system
	DUCKDB_API static string PathSeparator();
	//! Checks if path is starts with separator (i.e., '/' on UNIX '\\' on Windows)
	DUCKDB_API static bool IsPathAbsolute(const string &path);
	//! Normalize an absolute path - the goal of normalizing is converting "\test.db" and "C:/test.db" into "C:\test.db"
	//! so that the database system cache can correctly
	DUCKDB_API static string NormalizeAbsolutePath(const string &path);
	//! Join two paths together
	DUCKDB_API static string JoinPath(const string &a, const string &path);
	//! Convert separators in a path to the local separators (e.g. convert "/" into \\ on windows)
	DUCKDB_API static string ConvertSeparators(const string &path);
	//! Extract the base name of a file (e.g. if the input is lib/example.dll the base name is 'example')
	DUCKDB_API static string ExtractBaseName(const string &path);
	//! Extract the name of a file (e.g if the input is lib/example.dll the name is 'example.dll')
	DUCKDB_API static string ExtractName(const string &path);

	//! Returns the value of an environment variable - or the empty string if it is not set
	DUCKDB_API static string GetEnvVariable(const string &name);

	//! Whether there is a glob in the string
	DUCKDB_API static bool HasGlob(const string &str);
	//! Runs a glob on the file system, returning a list of matching files
	DUCKDB_API virtual vector<string> Glob(const string &path, FileOpener *opener = nullptr);
	DUCKDB_API vector<string> GlobFiles(const string &path, ClientContext &context,
	                                    FileGlobOptions options = FileGlobOptions::DISALLOW_EMPTY);

	//! registers a sub-file system to handle certain file name prefixes, e.g. http:// etc.
	DUCKDB_API virtual void RegisterSubSystem(unique_ptr<FileSystem> sub_fs);
	DUCKDB_API virtual void RegisterSubSystem(FileCompressionType compression_type, unique_ptr<FileSystem> fs);

	//! Unregister a sub-filesystem by name
	DUCKDB_API virtual void UnregisterSubSystem(const string &name);

	//! List registered sub-filesystems, including builtin ones
	DUCKDB_API virtual vector<string> ListSubSystems();

	//! Whether or not a sub-system can handle a specific file path
	DUCKDB_API virtual bool CanHandleFile(const string &fpath);

	//! Set the file pointer of a file handle to a specified location. Reads and writes will happen from this location
	DUCKDB_API virtual void Seek(FileHandle &handle, idx_t location);
	//! Reset a file to the beginning (equivalent to Seek(handle, 0) for simple files)
	DUCKDB_API virtual void Reset(FileHandle &handle);
	DUCKDB_API virtual idx_t SeekPosition(FileHandle &handle);

	//! Whether or not we can seek into the file
	DUCKDB_API virtual bool CanSeek();
	//! Whether or not the FS handles plain files on disk. This is relevant for certain optimizations, as random reads
	//! in a file on-disk are much cheaper than e.g. random reads in a file over the network
	DUCKDB_API virtual bool OnDiskFile(FileHandle &handle);

	DUCKDB_API virtual unique_ptr<FileHandle> OpenCompressedFile(unique_ptr<FileHandle> handle, bool write);

	//! Create a LocalFileSystem.
	DUCKDB_API static unique_ptr<FileSystem> CreateLocal();

	//! Return the name of the filesytem. Used for forming diagnosis messages.
	DUCKDB_API virtual std::string GetName() const = 0;

	//! Whether or not a file is remote or local, based only on file path
	DUCKDB_API static bool IsRemoteFile(const string &path);
};

} // namespace duckdb


namespace duckdb {

#define FILE_BUFFER_SIZE 4096

class BufferedFileWriter : public Serializer {
public:
	static constexpr uint8_t DEFAULT_OPEN_FLAGS = FileFlags::FILE_FLAGS_WRITE | FileFlags::FILE_FLAGS_FILE_CREATE;

	//! Serializes to a buffer allocated by the serializer, will expand when
	//! writing past the initial threshold
	DUCKDB_API BufferedFileWriter(FileSystem &fs, const string &path, uint8_t open_flags = DEFAULT_OPEN_FLAGS);

	FileSystem &fs;
	string path;
	unsafe_unique_array<data_t> data;
	idx_t offset;
	idx_t total_written;
	unique_ptr<FileHandle> handle;

public:
	DUCKDB_API void WriteData(const_data_ptr_t buffer, uint64_t write_size) override;
	//! Flush the buffer to disk and sync the file to ensure writing is completed
	DUCKDB_API void Sync();
	//! Flush the buffer to the file (without sync)
	DUCKDB_API void Flush();
	//! Returns the current size of the file
	DUCKDB_API int64_t GetFileSize();
	//! Truncate the size to a previous size (given that size <= GetFileSize())
	DUCKDB_API void Truncate(int64_t size);

	DUCKDB_API idx_t GetTotalWritten();
};

} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/function/udf_function.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/function/scalar_function.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/vector_operations/binary_executor.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/types/vector.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/bitset.hpp
//
//
//===----------------------------------------------------------------------===//



#include <bitset>

namespace duckdb {
using std::bitset;
}


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/vector_type.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

enum class VectorType : uint8_t {
	FLAT_VECTOR,       // Flat vectors represent a standard uncompressed vector
	FSST_VECTOR,       // Contains string data compressed with FSST
	CONSTANT_VECTOR,   // Constant vector represents a single constant
	DICTIONARY_VECTOR, // Dictionary vector represents a selection vector on top of another vector
	SEQUENCE_VECTOR    // Sequence vector represents a sequence with a start point and an increment
};

string VectorTypeToString(VectorType type);

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/types/selection_vector.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/types.hpp
//
//
//===----------------------------------------------------------------------===//








#include <limits>

namespace duckdb {

class FormatSerializer;
class FormatDeserializer;
class Serializer;
class Deserializer;
class Value;
class TypeCatalogEntry;
class Vector;
class ClientContext;
class FieldWriter;

//! Extra Type Info Type
enum class ExtraTypeInfoType : uint8_t {
	INVALID_TYPE_INFO = 0,
	GENERIC_TYPE_INFO = 1,
	DECIMAL_TYPE_INFO = 2,
	STRING_TYPE_INFO = 3,
	LIST_TYPE_INFO = 4,
	STRUCT_TYPE_INFO = 5,
	ENUM_TYPE_INFO = 6,
	USER_TYPE_INFO = 7,
	AGGREGATE_STATE_TYPE_INFO = 8
};

struct string_t;

template <class T>
using child_list_t = vector<std::pair<std::string, T>>;
//! FIXME: this should be a single_thread_ptr
template <class T>
using buffer_ptr = shared_ptr<T>;

template <class T, typename... Args>
buffer_ptr<T> make_buffer(Args &&...args) {
	return make_shared<T>(std::forward<Args>(args)...);
}

struct list_entry_t {
	list_entry_t() = default;
	list_entry_t(uint64_t offset, uint64_t length) : offset(offset), length(length) {
	}
	inline constexpr bool operator != (const list_entry_t &other) const {
		return !(*this == other);
	}
	inline constexpr bool operator == (const list_entry_t &other) const {
		return offset == other.offset && length == other.length;
	}

	uint64_t offset;
	uint64_t length;
};

using union_tag_t = uint8_t; 

//===--------------------------------------------------------------------===//
// Internal Types
//===--------------------------------------------------------------------===//

// taken from arrow's type.h
enum class PhysicalType : uint8_t {
	///// A NULL type having no physical storage
	//NA = 0,

	/// Boolean as 8 bit "bool" value
	BOOL = 1,

	/// Unsigned 8-bit little-endian integer
	UINT8 = 2,

	/// Signed 8-bit little-endian integer
	INT8 = 3,

	/// Unsigned 16-bit little-endian integer
	UINT16 = 4,

	/// Signed 16-bit little-endian integer
	INT16 = 5,

	/// Unsigned 32-bit little-endian integer
	UINT32 = 6,

	/// Signed 32-bit little-endian integer
	INT32 = 7,

	/// Unsigned 64-bit little-endian integer
	UINT64 = 8,

	/// Signed 64-bit little-endian integer
	INT64 = 9,

	///// 2-byte floating point value
	//HALF_FLOAT = 10,

	/// 4-byte floating point value
	FLOAT = 11,

	/// 8-byte floating point value
	DOUBLE = 12,

	///// UTF8 variable-length string as List<Char>
	//STRING = 13,

	///// Variable-length bytes (no guarantee of UTF8-ness)
	//BINARY = 14,

	///// Fixed-size binary. Each value occupies the same number of bytes
	//FIXED_SIZE_BINARY = 15,

	///// int32_t days since the UNIX epoch
	//DATE32 = 16,

	///// int64_t milliseconds since the UNIX epoch
	//DATE64 = 17,

	///// Exact timestamp encoded with int64 since UNIX epoch
	///// Default unit millisecond
	//TIMESTAMP = 18,

	///// Time as signed 32-bit integer, representing either seconds or
	///// milliseconds since midnight
	//TIME32 = 19,

	///// Time as signed 64-bit integer, representing either microseconds or
	///// nanoseconds since midnight
	//TIME64 = 20,

	/// YEAR_MONTH or DAY_TIME interval in SQL style
	INTERVAL = 21,

	/// Precision- and scale-based decimal type. Storage type depends on the
	/// parameters.
	// DECIMAL = 22,

	/// A list of some logical data type
	LIST = 23,

	/// Struct of logical types
	STRUCT = 24,

	///// Unions of logical types
	//UNION = 25,

	///// Dictionary-encoded type, also called "categorical" or "factor"
	///// in other programming languages. Holds the dictionary value
	///// type but not the dictionary itself, which is part of the
	///// ArrayData struct
	//DICTIONARY = 26,

	///// Custom data type, implemented by user
	//EXTENSION = 28,

	///// Fixed size list of some logical type
	//FIXED_SIZE_LIST = 29,

	///// Measure of elapsed time in either seconds, milliseconds, microseconds
	///// or nanoseconds.
	//DURATION = 30,

	///// Like STRING, but with 64-bit offsets
	//LARGE_STRING = 31,

	///// Like BINARY, but with 64-bit offsets
	//LARGE_BINARY = 32,

	///// Like LIST, but with 64-bit offsets
	//LARGE_LIST = 33,

	/// DuckDB Extensions
	VARCHAR = 200, // our own string representation, different from STRING and LARGE_STRING above
	INT128 = 204, // 128-bit integers
	UNKNOWN = 205, // Unknown physical type of user defined types
	/// Boolean as 1 bit, LSB bit-packed ordering
	BIT = 206,

	INVALID = 255
};

//===--------------------------------------------------------------------===//
// SQL Types
//===--------------------------------------------------------------------===//
enum class LogicalTypeId : uint8_t {
	INVALID = 0,
	SQLNULL = 1, /* NULL type, used for constant NULL */
	UNKNOWN = 2, /* unknown type, used for parameter expressions */
	ANY = 3,     /* ANY type, used for functions that accept any type as parameter */
	USER = 4, /* A User Defined Type (e.g., ENUMs before the binder) */
	BOOLEAN = 10,
	TINYINT = 11,
	SMALLINT = 12,
	INTEGER = 13,
	BIGINT = 14,
	DATE = 15,
	TIME = 16,
	TIMESTAMP_SEC = 17,
	TIMESTAMP_MS = 18,
	TIMESTAMP = 19, //! us
	TIMESTAMP_NS = 20,
	DECIMAL = 21,
	FLOAT = 22,
	DOUBLE = 23,
	CHAR = 24,
	VARCHAR = 25,
	BLOB = 26,
	INTERVAL = 27,
	UTINYINT = 28,
	USMALLINT = 29,
	UINTEGER = 30,
	UBIGINT = 31,
	TIMESTAMP_TZ = 32,
	TIME_TZ = 34,
	BIT = 36,

	HUGEINT = 50,
	POINTER = 51,
	VALIDITY = 53,
	UUID = 54,

	STRUCT = 100,
	LIST = 101,
	MAP = 102,
	TABLE = 103,
	ENUM = 104,
	AGGREGATE_STATE = 105,
	LAMBDA = 106,
	UNION = 107
};


struct ExtraTypeInfo;

struct aggregate_state_t;

struct LogicalType {
	DUCKDB_API LogicalType();
	DUCKDB_API LogicalType(LogicalTypeId id); // NOLINT: Allow implicit conversion from `LogicalTypeId`
	DUCKDB_API LogicalType(LogicalTypeId id, shared_ptr<ExtraTypeInfo> type_info);
	DUCKDB_API LogicalType(const LogicalType &other);
	DUCKDB_API LogicalType(LogicalType &&other) noexcept;

	DUCKDB_API ~LogicalType();

	inline LogicalTypeId id() const {
		return id_;
	}
	inline PhysicalType InternalType() const {
		return physical_type_;
	}
	inline const ExtraTypeInfo *AuxInfo() const {
		return type_info_.get();
	}

	inline shared_ptr<ExtraTypeInfo> GetAuxInfoShrPtr() const {
		return type_info_;
	}

	inline void CopyAuxInfo(const LogicalType& other) {
		type_info_ = other.type_info_;
	}
	bool EqualTypeInfo(const LogicalType& rhs) const;

	// copy assignment
	inline LogicalType& operator=(const LogicalType &other) {
		id_ = other.id_;
		physical_type_ = other.physical_type_;
		type_info_ = other.type_info_;
		return *this;
	}
	// move assignment
	inline LogicalType& operator=(LogicalType&& other) noexcept {
		id_ = other.id_;
		physical_type_ = other.physical_type_;
		std::swap(type_info_, other.type_info_);
		return *this;
	}

	DUCKDB_API bool operator==(const LogicalType &rhs) const;
	inline bool operator!=(const LogicalType &rhs) const {
		return !(*this == rhs);
	}

	//! Serializes a LogicalType to a stand-alone binary blob
	DUCKDB_API void Serialize(Serializer &serializer) const;

	DUCKDB_API void SerializeEnumType(Serializer &serializer) const;

	//! Deserializes a blob back into an LogicalType
	DUCKDB_API static LogicalType Deserialize(Deserializer &source);

	DUCKDB_API void FormatSerialize(FormatSerializer &serializer) const;
	DUCKDB_API static LogicalType FormatDeserialize(FormatDeserializer &deserializer);


	static bool TypeIsTimestamp(LogicalTypeId id) {
		return (id == LogicalTypeId::TIMESTAMP ||
				id == LogicalTypeId::TIMESTAMP_MS ||
				id == LogicalTypeId::TIMESTAMP_NS ||
				id == LogicalTypeId::TIMESTAMP_SEC ||
				id == LogicalTypeId::TIMESTAMP_TZ);
	}
	static bool TypeIsTimestamp(const LogicalType& type) {
		return TypeIsTimestamp(type.id());
	}
	DUCKDB_API string ToString() const;
	DUCKDB_API bool IsIntegral() const;
	DUCKDB_API bool IsNumeric() const;
	DUCKDB_API hash_t Hash() const;
	DUCKDB_API void SetAlias(string alias);
	DUCKDB_API bool HasAlias() const;
	DUCKDB_API string GetAlias() const;

	DUCKDB_API static LogicalType MaxLogicalType(const LogicalType &left, const LogicalType &right);

	DUCKDB_API static ExtraTypeInfoType GetExtraTypeInfoType(const ExtraTypeInfo &type);

	//! Gets the decimal properties of a numeric type. Fails if the type is not numeric.
	DUCKDB_API bool GetDecimalProperties(uint8_t &width, uint8_t &scale) const;

	DUCKDB_API void Verify() const;

	DUCKDB_API bool IsValid() const;

private:
	LogicalTypeId id_;
	PhysicalType physical_type_;
	shared_ptr<ExtraTypeInfo> type_info_;

private:
	PhysicalType GetInternalType();

public:
	static constexpr const LogicalTypeId SQLNULL = LogicalTypeId::SQLNULL;
	static constexpr const LogicalTypeId UNKNOWN = LogicalTypeId::UNKNOWN;
	static constexpr const LogicalTypeId BOOLEAN = LogicalTypeId::BOOLEAN;
	static constexpr const LogicalTypeId TINYINT = LogicalTypeId::TINYINT;
	static constexpr const LogicalTypeId UTINYINT = LogicalTypeId::UTINYINT;
	static constexpr const LogicalTypeId SMALLINT = LogicalTypeId::SMALLINT;
	static constexpr const LogicalTypeId USMALLINT = LogicalTypeId::USMALLINT;
	static constexpr const LogicalTypeId INTEGER = LogicalTypeId::INTEGER;
	static constexpr const LogicalTypeId UINTEGER = LogicalTypeId::UINTEGER;
	static constexpr const LogicalTypeId BIGINT = LogicalTypeId::BIGINT;
	static constexpr const LogicalTypeId UBIGINT = LogicalTypeId::UBIGINT;
	static constexpr const LogicalTypeId FLOAT = LogicalTypeId::FLOAT;
	static constexpr const LogicalTypeId DOUBLE = LogicalTypeId::DOUBLE;
	static constexpr const LogicalTypeId DATE = LogicalTypeId::DATE;
	static constexpr const LogicalTypeId TIMESTAMP = LogicalTypeId::TIMESTAMP;
	static constexpr const LogicalTypeId TIMESTAMP_S = LogicalTypeId::TIMESTAMP_SEC;
	static constexpr const LogicalTypeId TIMESTAMP_MS = LogicalTypeId::TIMESTAMP_MS;
	static constexpr const LogicalTypeId TIMESTAMP_NS = LogicalTypeId::TIMESTAMP_NS;
	static constexpr const LogicalTypeId TIME = LogicalTypeId::TIME;
	static constexpr const LogicalTypeId TIMESTAMP_TZ = LogicalTypeId::TIMESTAMP_TZ;
	static constexpr const LogicalTypeId TIME_TZ = LogicalTypeId::TIME_TZ;
	static constexpr const LogicalTypeId VARCHAR = LogicalTypeId::VARCHAR;
	static constexpr const LogicalTypeId ANY = LogicalTypeId::ANY;
	static constexpr const LogicalTypeId BLOB = LogicalTypeId::BLOB;
    static constexpr const LogicalTypeId BIT = LogicalTypeId::BIT;
    static constexpr const LogicalTypeId INTERVAL = LogicalTypeId::INTERVAL;
	static constexpr const LogicalTypeId HUGEINT = LogicalTypeId::HUGEINT;
	static constexpr const LogicalTypeId UUID = LogicalTypeId::UUID;
	static constexpr const LogicalTypeId HASH = LogicalTypeId::UBIGINT;
	static constexpr const LogicalTypeId POINTER = LogicalTypeId::POINTER;
	static constexpr const LogicalTypeId TABLE = LogicalTypeId::TABLE;
	static constexpr const LogicalTypeId LAMBDA = LogicalTypeId::LAMBDA;
	static constexpr const LogicalTypeId INVALID = LogicalTypeId::INVALID;
	static constexpr const LogicalTypeId ROW_TYPE = LogicalTypeId::BIGINT;

	// explicitly allowing these functions to be capitalized to be in-line with the remaining functions
	DUCKDB_API static LogicalType DECIMAL(int width, int scale);                 // NOLINT
	DUCKDB_API static LogicalType VARCHAR_COLLATION(string collation);           // NOLINT
	DUCKDB_API static LogicalType LIST(const LogicalType &child);                       // NOLINT
	DUCKDB_API static LogicalType STRUCT(child_list_t<LogicalType> children);    // NOLINT
	DUCKDB_API static LogicalType AGGREGATE_STATE(aggregate_state_t state_type);    // NOLINT
	DUCKDB_API static LogicalType MAP(const LogicalType &child);				// NOLINT
	DUCKDB_API static LogicalType MAP( child_list_t<LogicalType> children);       // NOLINT
	DUCKDB_API static LogicalType MAP(LogicalType key, LogicalType value); // NOLINT
	DUCKDB_API static LogicalType UNION( child_list_t<LogicalType> members);     // NOLINT
	DUCKDB_API static LogicalType ENUM(const string &enum_name, Vector &ordered_data, idx_t size); // NOLINT
	DUCKDB_API static LogicalType USER(const string &user_type_name); // NOLINT
	//! A list of all NUMERIC types (integral and floating point types)
	DUCKDB_API static const vector<LogicalType> Numeric();
	//! A list of all INTEGRAL types
	DUCKDB_API static const vector<LogicalType> Integral();
	//! A list of ALL SQL types
	DUCKDB_API static const vector<LogicalType> AllTypes();
};

struct DecimalType {
	DUCKDB_API static uint8_t GetWidth(const LogicalType &type);
	DUCKDB_API static uint8_t GetScale(const LogicalType &type);
	DUCKDB_API static uint8_t MaxWidth();
};

struct StringType {
	DUCKDB_API static string GetCollation(const LogicalType &type);
};

struct ListType {
	DUCKDB_API static const LogicalType &GetChildType(const LogicalType &type);
};

struct UserType{
	DUCKDB_API static const string &GetTypeName(const LogicalType &type);
};

struct EnumType{
	DUCKDB_API static const string &GetTypeName(const LogicalType &type);
	DUCKDB_API static int64_t GetPos(const LogicalType &type, const string_t& key);
	DUCKDB_API static const Vector &GetValuesInsertOrder(const LogicalType &type);
	DUCKDB_API static idx_t GetSize(const LogicalType &type);
	DUCKDB_API static const string GetValue(const Value &val);
	DUCKDB_API static void SetCatalog(LogicalType &type, optional_ptr<TypeCatalogEntry> catalog_entry);
	DUCKDB_API static optional_ptr<TypeCatalogEntry> GetCatalog(const LogicalType &type);
	DUCKDB_API static string GetSchemaName(const LogicalType &type);
	DUCKDB_API static PhysicalType GetPhysicalType(const LogicalType &type);
	DUCKDB_API static void Serialize(FieldWriter& writer, const ExtraTypeInfo& type_info, bool serialize_internals);
};

struct StructType {
	DUCKDB_API static const child_list_t<LogicalType> &GetChildTypes(const LogicalType &type);
	DUCKDB_API static const LogicalType &GetChildType(const LogicalType &type, idx_t index);
	DUCKDB_API static const string &GetChildName(const LogicalType &type, idx_t index);
	DUCKDB_API static idx_t GetChildCount(const LogicalType &type);
};

struct MapType {
	DUCKDB_API static const LogicalType &KeyType(const LogicalType &type);
	DUCKDB_API static const LogicalType &ValueType(const LogicalType &type);
};

struct UnionType {
	DUCKDB_API static const idx_t MAX_UNION_MEMBERS = 256;
	DUCKDB_API static idx_t GetMemberCount(const LogicalType &type);
	DUCKDB_API static const LogicalType &GetMemberType(const LogicalType &type, idx_t index);
	DUCKDB_API static const string &GetMemberName(const LogicalType &type, idx_t index);
	DUCKDB_API static const child_list_t<LogicalType> CopyMemberTypes(const LogicalType &type);
};

struct AggregateStateType {
	DUCKDB_API static const string GetTypeName(const LogicalType &type);
	DUCKDB_API static const aggregate_state_t &GetStateType(const LogicalType &type);
};

// **DEPRECATED**: Use EnumUtil directly instead.
DUCKDB_API string LogicalTypeIdToString(LogicalTypeId type);

DUCKDB_API LogicalTypeId TransformStringToLogicalTypeId(const string &str);

DUCKDB_API LogicalType TransformStringToLogicalType(const string &str);

DUCKDB_API LogicalType TransformStringToLogicalType(const string &str, ClientContext &context);

//! The PhysicalType used by the row identifiers column
extern const PhysicalType ROW_TYPE;

DUCKDB_API string TypeIdToString(PhysicalType type);
DUCKDB_API idx_t GetTypeIdSize(PhysicalType type);
DUCKDB_API bool TypeIsConstantSize(PhysicalType type);
DUCKDB_API bool TypeIsIntegral(PhysicalType type);
DUCKDB_API bool TypeIsNumeric(PhysicalType type);
DUCKDB_API bool TypeIsInteger(PhysicalType type);

bool ApproxEqual(float l, float r);
bool ApproxEqual(double l, double r);

struct aggregate_state_t {
	aggregate_state_t(string function_name_p, LogicalType return_type_p, vector<LogicalType> bound_argument_types_p) : function_name(std::move(function_name_p)), return_type(std::move(return_type_p)), bound_argument_types(std::move(bound_argument_types_p)) {
	}

	string function_name;
	LogicalType return_type;
	vector<LogicalType> bound_argument_types;
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/vector_size.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

//! The vector size used in the execution engine
#ifndef STANDARD_VECTOR_SIZE
#define STANDARD_VECTOR_SIZE 2048
#endif

#if ((STANDARD_VECTOR_SIZE & (STANDARD_VECTOR_SIZE - 1)) != 0)
#error Vector size should be a power of two
#endif

//! Zero selection vector: completely filled with the value 0 [READ ONLY]
extern const sel_t ZERO_VECTOR[STANDARD_VECTOR_SIZE];

} // namespace duckdb


namespace duckdb {
class VectorBuffer;

struct SelectionData {
	DUCKDB_API explicit SelectionData(idx_t count);

	unsafe_unique_array<sel_t> owned_data;
};

struct SelectionVector {
	SelectionVector() : sel_vector(nullptr) {
	}
	explicit SelectionVector(sel_t *sel) {
		Initialize(sel);
	}
	explicit SelectionVector(idx_t count) {
		Initialize(count);
	}
	SelectionVector(idx_t start, idx_t count) {
		Initialize(STANDARD_VECTOR_SIZE);
		for (idx_t i = 0; i < count; i++) {
			set_index(i, start + i);
		}
	}
	SelectionVector(const SelectionVector &sel_vector) {
		Initialize(sel_vector);
	}
	explicit SelectionVector(buffer_ptr<SelectionData> data) {
		Initialize(std::move(data));
	}
	SelectionVector &operator=(SelectionVector &&other) {
		sel_vector = other.sel_vector;
		other.sel_vector = nullptr;
		selection_data = std::move(other.selection_data);
		return *this;
	}

public:
	static idx_t Inverted(const SelectionVector &src, SelectionVector &dst, idx_t source_size, idx_t count) {
		idx_t src_idx = 0;
		idx_t dst_idx = 0;
		for (idx_t i = 0; i < count; i++) {
			if (src_idx < source_size && src.get_index(src_idx) == i) {
				src_idx++;
				// This index is selected by 'src', skip it in 'dst'
				continue;
			}
			// This index does not exist in 'src', add it to the selection of 'dst'
			dst.set_index(dst_idx++, i);
		}
		return dst_idx;
	}

	void Initialize(sel_t *sel) {
		selection_data.reset();
		sel_vector = sel;
	}
	void Initialize(idx_t count = STANDARD_VECTOR_SIZE) {
		selection_data = make_shared<SelectionData>(count);
		sel_vector = selection_data->owned_data.get();
	}
	void Initialize(buffer_ptr<SelectionData> data) {
		selection_data = std::move(data);
		sel_vector = selection_data->owned_data.get();
	}
	void Initialize(const SelectionVector &other) {
		selection_data = other.selection_data;
		sel_vector = other.sel_vector;
	}

	inline void set_index(idx_t idx, idx_t loc) {
		sel_vector[idx] = loc;
	}
	inline void swap(idx_t i, idx_t j) {
		sel_t tmp = sel_vector[i];
		sel_vector[i] = sel_vector[j];
		sel_vector[j] = tmp;
	}
	inline idx_t get_index(idx_t idx) const {
		return sel_vector ? sel_vector[idx] : idx;
	}
	sel_t *data() {
		return sel_vector;
	}
	const sel_t *data() const {
		return sel_vector;
	}
	buffer_ptr<SelectionData> sel_data() {
		return selection_data;
	}
	buffer_ptr<SelectionData> Slice(const SelectionVector &sel, idx_t count) const;

	string ToString(idx_t count = 0) const;
	void Print(idx_t count = 0) const;

	inline sel_t &operator[](idx_t index) const {
		return sel_vector[index];
	}

private:
	sel_t *sel_vector;
	buffer_ptr<SelectionData> selection_data;
};

class OptionalSelection {
public:
	explicit inline OptionalSelection(SelectionVector *sel_p) {
		Initialize(sel_p);
	}
	void Initialize(SelectionVector *sel_p) {
		sel = sel_p;
		if (sel) {
			vec.Initialize(sel->data());
			sel = &vec;
		}
	}

	inline operator SelectionVector *() {
		return sel;
	}

	inline void Append(idx_t &count, const idx_t idx) {
		if (sel) {
			sel->set_index(count, idx);
		}
		++count;
	}

	inline void Advance(idx_t completed) {
		if (sel) {
			sel->Initialize(sel->data() + completed);
		}
	}

private:
	SelectionVector *sel;
	SelectionVector vec;
};

// Contains a selection vector, combined with a count
class ManagedSelection {
public:
	explicit inline ManagedSelection(idx_t size, bool initialize = true)
	    : initialized(initialize), size(size), internal_opt_selvec(nullptr) {
		count = 0;
		if (!initialized) {
			return;
		}
		sel_vec.Initialize(size);
		internal_opt_selvec.Initialize(&sel_vec);
	}

public:
	bool Initialized() const {
		return initialized;
	}
	void Initialize(idx_t size) {
		D_ASSERT(!initialized);
		this->size = size;
		sel_vec.Initialize(size);
		internal_opt_selvec.Initialize(&sel_vec);
		initialized = true;
	}

	inline idx_t operator[](idx_t index) const {
		D_ASSERT(index < size);
		return sel_vec.get_index(index);
	}
	inline bool IndexMapsToLocation(idx_t idx, idx_t location) const {
		return idx < count && sel_vec.get_index(idx) == location;
	}
	inline void Append(const idx_t idx) {
		internal_opt_selvec.Append(count, idx);
	}
	inline idx_t Count() const {
		return count;
	}
	inline idx_t Size() const {
		return size;
	}
	inline const SelectionVector &Selection() const {
		return sel_vec;
	}
	inline SelectionVector &Selection() {
		return sel_vec;
	}

private:
	bool initialized = false;
	idx_t count;
	idx_t size;
	SelectionVector sel_vec;
	OptionalSelection internal_opt_selvec;
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/types/validity_mask.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/to_string.hpp
//
//
//===----------------------------------------------------------------------===//



namespace duckdb {
using std::to_string;
}




namespace duckdb {
struct ValidityMask;

template <typename V>
struct TemplatedValidityData {
	static constexpr const int BITS_PER_VALUE = sizeof(V) * 8;
	static constexpr const V MAX_ENTRY = ~V(0);

public:
	inline explicit TemplatedValidityData(idx_t count) {
		auto entry_count = EntryCount(count);
		owned_data = make_unsafe_uniq_array<V>(entry_count);
		for (idx_t entry_idx = 0; entry_idx < entry_count; entry_idx++) {
			owned_data[entry_idx] = MAX_ENTRY;
		}
	}
	inline TemplatedValidityData(const V *validity_mask, idx_t count) {
		D_ASSERT(validity_mask);
		auto entry_count = EntryCount(count);
		owned_data = make_unsafe_uniq_array<V>(entry_count);
		for (idx_t entry_idx = 0; entry_idx < entry_count; entry_idx++) {
			owned_data[entry_idx] = validity_mask[entry_idx];
		}
	}

	unsafe_unique_array<V> owned_data;

public:
	static inline idx_t EntryCount(idx_t count) {
		return (count + (BITS_PER_VALUE - 1)) / BITS_PER_VALUE;
	}
};

using validity_t = uint64_t;

struct ValidityData : TemplatedValidityData<validity_t> {
public:
	DUCKDB_API explicit ValidityData(idx_t count);
	DUCKDB_API ValidityData(const ValidityMask &original, idx_t count);
};

//! Type used for validity masks
template <typename V>
struct TemplatedValidityMask {
	using ValidityBuffer = TemplatedValidityData<V>;

public:
	static constexpr const int BITS_PER_VALUE = ValidityBuffer::BITS_PER_VALUE;
	static constexpr const int STANDARD_ENTRY_COUNT = (STANDARD_VECTOR_SIZE + (BITS_PER_VALUE - 1)) / BITS_PER_VALUE;
	static constexpr const int STANDARD_MASK_SIZE = STANDARD_ENTRY_COUNT * sizeof(validity_t);

public:
	inline TemplatedValidityMask() : validity_mask(nullptr) {
	}
	inline explicit TemplatedValidityMask(idx_t max_count) {
		Initialize(max_count);
	}
	inline explicit TemplatedValidityMask(V *ptr) : validity_mask(ptr) {
	}
	inline TemplatedValidityMask(const TemplatedValidityMask &original, idx_t count) {
		Copy(original, count);
	}

	static inline idx_t ValidityMaskSize(idx_t count = STANDARD_VECTOR_SIZE) {
		return ValidityBuffer::EntryCount(count) * sizeof(V);
	}
	inline bool AllValid() const {
		return !validity_mask;
	}
	inline bool CheckAllValid(idx_t count) const {
		return CountValid(count) == count;
	}

	inline bool CheckAllValid(idx_t to, idx_t from) const {
		if (AllValid()) {
			return true;
		}
		for (idx_t i = from; i < to; i++) {
			if (!RowIsValid(i)) {
				return false;
			}
		}
		return true;
	}

	idx_t CountValid(const idx_t count) const {
		if (AllValid() || count == 0) {
			return count;
		}

		idx_t valid = 0;
		const auto entry_count = EntryCount(count);
		for (idx_t entry_idx = 0; entry_idx < entry_count;) {
			auto entry = GetValidityEntry(entry_idx++);
			// Handle ragged end (if not exactly multiple of BITS_PER_VALUE)
			if (entry_idx == entry_count && count % BITS_PER_VALUE != 0) {
				idx_t idx_in_entry;
				GetEntryIndex(count, entry_idx, idx_in_entry);
				for (idx_t i = 0; i < idx_in_entry; ++i) {
					valid += idx_t(RowIsValid(entry, i));
				}
				break;
			}

			// Handle all set
			if (AllValid(entry)) {
				valid += BITS_PER_VALUE;
				continue;
			}

			// Count partial entry (Kernighan's algorithm)
			while (entry) {
				entry &= (entry - 1);
				++valid;
			}
		}

		return valid;
	}

	inline V *GetData() const {
		return validity_mask;
	}
	inline void Reset() {
		validity_mask = nullptr;
		validity_data.reset();
	}

	static inline idx_t EntryCount(idx_t count) {
		return ValidityBuffer::EntryCount(count);
	}
	inline V GetValidityEntry(idx_t entry_idx) const {
		if (!validity_mask) {
			return ValidityBuffer::MAX_ENTRY;
		}
		return validity_mask[entry_idx];
	}
	static inline bool AllValid(V entry) {
		return entry == ValidityBuffer::MAX_ENTRY;
	}
	static inline bool NoneValid(V entry) {
		return entry == 0;
	}
	static inline bool RowIsValid(V entry, idx_t idx_in_entry) {
		return entry & (V(1) << V(idx_in_entry));
	}
	static inline void GetEntryIndex(idx_t row_idx, idx_t &entry_idx, idx_t &idx_in_entry) {
		entry_idx = row_idx / BITS_PER_VALUE;
		idx_in_entry = row_idx % BITS_PER_VALUE;
	}
	//! Get an entry that has first-n bits set as valid and rest set as invalid
	static inline V EntryWithValidBits(idx_t n) {
		if (n == 0) {
			return V(0);
		}
		return ValidityBuffer::MAX_ENTRY >> (BITS_PER_VALUE - n);
	}
	static inline idx_t SizeInBytes(idx_t n) {
		return (n + BITS_PER_VALUE - 1) / BITS_PER_VALUE;
	}

	//! RowIsValidUnsafe should only be used if AllValid() is false: it achieves the same as RowIsValid but skips a
	//! not-null check
	inline bool RowIsValidUnsafe(idx_t row_idx) const {
		D_ASSERT(validity_mask);
		idx_t entry_idx, idx_in_entry;
		GetEntryIndex(row_idx, entry_idx, idx_in_entry);
		auto entry = GetValidityEntry(entry_idx);
		return RowIsValid(entry, idx_in_entry);
	}

	//! Returns true if a row is valid (i.e. not null), false otherwise
	inline bool RowIsValid(idx_t row_idx) const {
		if (!validity_mask) {
			return true;
		}
		return RowIsValidUnsafe(row_idx);
	}

	//! Same as SetValid, but skips a null check on validity_mask
	inline void SetValidUnsafe(idx_t row_idx) {
		D_ASSERT(validity_mask);
		idx_t entry_idx, idx_in_entry;
		GetEntryIndex(row_idx, entry_idx, idx_in_entry);
		validity_mask[entry_idx] |= (V(1) << V(idx_in_entry));
	}

	//! Marks the entry at the specified row index as valid (i.e. not-null)
	inline void SetValid(idx_t row_idx) {
		if (!validity_mask) {
			// if AllValid() we don't need to do anything
			// the row is already valid
			return;
		}
		SetValidUnsafe(row_idx);
	}

	//! Marks the bit at the specified entry as invalid (i.e. null)
	inline void SetInvalidUnsafe(idx_t entry_idx, idx_t idx_in_entry) {
		D_ASSERT(validity_mask);
		validity_mask[entry_idx] &= ~(V(1) << V(idx_in_entry));
	}

	//! Marks the bit at the specified row index as invalid (i.e. null)
	inline void SetInvalidUnsafe(idx_t row_idx) {
		idx_t entry_idx, idx_in_entry;
		GetEntryIndex(row_idx, entry_idx, idx_in_entry);
		SetInvalidUnsafe(entry_idx, idx_in_entry);
	}

	//! Marks the entry at the specified row index as invalid (i.e. null)
	inline void SetInvalid(idx_t row_idx) {
		if (!validity_mask) {
			D_ASSERT(row_idx <= STANDARD_VECTOR_SIZE);
			Initialize(STANDARD_VECTOR_SIZE);
		}
		SetInvalidUnsafe(row_idx);
	}

	//! Mark the entry at the specified index as either valid or invalid (non-null or null)
	inline void Set(idx_t row_idx, bool valid) {
		if (valid) {
			SetValid(row_idx);
		} else {
			SetInvalid(row_idx);
		}
	}

	//! Ensure the validity mask is writable, allocating space if it is not initialized
	inline void EnsureWritable() {
		if (!validity_mask) {
			Initialize();
		}
	}

	//! Marks exactly "count" bits in the validity mask as invalid (null)
	inline void SetAllInvalid(idx_t count) {
		EnsureWritable();
		if (count == 0) {
			return;
		}
		auto last_entry_index = ValidityBuffer::EntryCount(count) - 1;
		for (idx_t i = 0; i < last_entry_index; i++) {
			validity_mask[i] = 0;
		}
		auto last_entry_bits = count % static_cast<idx_t>(BITS_PER_VALUE);
		validity_mask[last_entry_index] = (last_entry_bits == 0) ? 0 : (ValidityBuffer::MAX_ENTRY << (last_entry_bits));
	}

	//! Marks exactly "count" bits in the validity mask as valid (not null)
	inline void SetAllValid(idx_t count) {
		EnsureWritable();
		if (count == 0) {
			return;
		}
		auto last_entry_index = ValidityBuffer::EntryCount(count) - 1;
		for (idx_t i = 0; i < last_entry_index; i++) {
			validity_mask[i] = ValidityBuffer::MAX_ENTRY;
		}
		auto last_entry_bits = count % static_cast<idx_t>(BITS_PER_VALUE);
		validity_mask[last_entry_index] |=
		    (last_entry_bits == 0) ? ValidityBuffer::MAX_ENTRY : ~(ValidityBuffer::MAX_ENTRY << (last_entry_bits));
	}

	inline bool IsMaskSet() const {
		if (validity_mask) {
			return true;
		}
		return false;
	}

public:
	inline void Initialize(validity_t *validity) {
		validity_data.reset();
		validity_mask = validity;
	}
	inline void Initialize(const TemplatedValidityMask &other) {
		validity_mask = other.validity_mask;
		validity_data = other.validity_data;
	}
	inline void Initialize(idx_t count = STANDARD_VECTOR_SIZE) {
		validity_data = make_buffer<ValidityBuffer>(count);
		validity_mask = validity_data->owned_data.get();
	}
	inline void Copy(const TemplatedValidityMask &other, idx_t count) {
		if (other.AllValid()) {
			validity_data = nullptr;
			validity_mask = nullptr;
		} else {
			validity_data = make_buffer<ValidityBuffer>(other.validity_mask, count);
			validity_mask = validity_data->owned_data.get();
		}
	}

protected:
	V *validity_mask;
	buffer_ptr<ValidityBuffer> validity_data;
};

struct ValidityMask : public TemplatedValidityMask<validity_t> {
public:
	inline ValidityMask() : TemplatedValidityMask(nullptr) {
	}
	inline explicit ValidityMask(idx_t max_count) : TemplatedValidityMask(max_count) {
	}
	inline explicit ValidityMask(validity_t *ptr) : TemplatedValidityMask(ptr) {
	}
	inline ValidityMask(const ValidityMask &original, idx_t count) : TemplatedValidityMask(original, count) {
	}

public:
	DUCKDB_API void Resize(idx_t old_size, idx_t new_size);

	DUCKDB_API void SliceInPlace(const ValidityMask &other, idx_t target_offset, idx_t source_offset, idx_t count);
	DUCKDB_API void Slice(const ValidityMask &other, idx_t source_offset, idx_t count);
	DUCKDB_API void Combine(const ValidityMask &other, idx_t count);
	DUCKDB_API string ToString(idx_t count) const;

	DUCKDB_API static bool IsAligned(idx_t count);
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/types/value.hpp
//
//
//===----------------------------------------------------------------------===//







//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/types/timestamp.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/limits.hpp
//
//
//===----------------------------------------------------------------------===//





// Undef annoying windows macro
#undef max

#include <limits>

namespace duckdb {

template <class T>
struct NumericLimits {
	static constexpr T Minimum() {
		return std::numeric_limits<T>::lowest();
	};
	static constexpr T Maximum() {
		return std::numeric_limits<T>::max();
	};
	DUCKDB_API static bool IsSigned();
	DUCKDB_API static idx_t Digits();
};

template <>
struct NumericLimits<int8_t> {
	static constexpr int8_t Minimum() {
		return std::numeric_limits<int8_t>::lowest();
	};
	static constexpr int8_t Maximum() {
		return std::numeric_limits<int8_t>::max();
	};
	static bool IsSigned() {
		return true;
	}
	static idx_t Digits() {
		return 3;
	}
};
template <>
struct NumericLimits<int16_t> {
	static constexpr int16_t Minimum() {
		return std::numeric_limits<int16_t>::lowest();
	};
	static constexpr int16_t Maximum() {
		return std::numeric_limits<int16_t>::max();
	};
	static bool IsSigned() {
		return true;
	}
	static idx_t Digits() {
		return 5;
	}
};
template <>
struct NumericLimits<int32_t> {
	static constexpr int32_t Minimum() {
		return std::numeric_limits<int32_t>::lowest();
	};
	static constexpr int32_t Maximum() {
		return std::numeric_limits<int32_t>::max();
	};
	static bool IsSigned() {
		return true;
	}
	static idx_t Digits() {
		return 10;
	}
};

template <>
struct NumericLimits<int64_t> {
	static constexpr int64_t Minimum() {
		return std::numeric_limits<int64_t>::lowest();
	};
	static constexpr int64_t Maximum() {
		return std::numeric_limits<int64_t>::max();
	};
	static bool IsSigned() {
		return true;
	}
	static idx_t Digits() {
		return 19;
	}
};
template <>
struct NumericLimits<hugeint_t> {
	static constexpr hugeint_t Minimum() {
		return {std::numeric_limits<int64_t>::lowest(), 1};
	};
	static constexpr hugeint_t Maximum() {
		return {std::numeric_limits<int64_t>::max(), std::numeric_limits<uint64_t>::max()};
	};
	static bool IsSigned() {
		return true;
	}
	static idx_t Digits() {
		return 39;
	}
};

template <>
struct NumericLimits<uint8_t> {
	static constexpr uint8_t Minimum() {
		return std::numeric_limits<uint8_t>::lowest();
	};
	static constexpr uint8_t Maximum() {
		return std::numeric_limits<uint8_t>::max();
	};
	static bool IsSigned() {
		return false;
	}
	static idx_t Digits() {
		return 3;
	}
};

template <>
struct NumericLimits<uint16_t> {
	static constexpr uint16_t Minimum() {
		return std::numeric_limits<uint16_t>::lowest();
	};
	static constexpr uint16_t Maximum() {
		return std::numeric_limits<uint16_t>::max();
	};
	static bool IsSigned() {
		return false;
	}
	static idx_t Digits() {
		return 5;
	}
};
template <>
struct NumericLimits<uint32_t> {
	static constexpr uint32_t Minimum() {
		return std::numeric_limits<uint32_t>::lowest();
	};
	static constexpr uint32_t Maximum() {
		return std::numeric_limits<uint32_t>::max();
	};
	static bool IsSigned() {
		return false;
	}
	static idx_t Digits() {
		return 10;
	}
};
template <>
struct NumericLimits<uint64_t> {
	static constexpr uint64_t Minimum() {
		return std::numeric_limits<uint64_t>::lowest();
	};
	static constexpr uint64_t Maximum() {
		return std::numeric_limits<uint64_t>::max();
	};
	static bool IsSigned() {
		return false;
	}
	static idx_t Digits() {
		return 20;
	}
};
template <>
struct NumericLimits<float> {
	static constexpr float Minimum() {
		return std::numeric_limits<float>::lowest();
	};
	static constexpr float Maximum() {
		return std::numeric_limits<float>::max();
	};
	static bool IsSigned() {
		return true;
	}
	static idx_t Digits() {
		return 127;
	}
};
template <>
struct NumericLimits<double> {
	static constexpr double Minimum() {
		return std::numeric_limits<double>::lowest();
	};
	static constexpr double Maximum() {
		return std::numeric_limits<double>::max();
	};
	static bool IsSigned() {
		return true;
	}
	static idx_t Digits() {
		return 250;
	}
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/string_util.hpp
//
//
//===----------------------------------------------------------------------===//







#include <cstring>

namespace duckdb {

/**
 * String Utility Functions
 * Note that these are not the most efficient implementations (i.e., they copy
 * memory) and therefore they should only be used for debug messages and other
 * such things.
 */
class StringUtil {
public:
	static string GenerateRandomName(idx_t length = 16);

	static uint8_t GetHexValue(char c) {
		if (c >= '0' && c <= '9') {
			return c - '0';
		}
		if (c >= 'a' && c <= 'f') {
			return c - 'a' + 10;
		}
		if (c >= 'A' && c <= 'F') {
			return c - 'A' + 10;
		}
		throw InvalidInputException("Invalid input for hex digit: %s", string(c, 1));
	}

	static uint8_t GetBinaryValue(char c) {
		if (c >= '0' && c <= '1') {
			return c - '0';
		}
		throw InvalidInputException("Invalid input for binary digit: %s", string(c, 1));
	}

	static bool CharacterIsSpace(char c) {
		return c == ' ' || c == '\t' || c == '\n' || c == '\v' || c == '\f' || c == '\r';
	}
	static bool CharacterIsNewline(char c) {
		return c == '\n' || c == '\r';
	}
	static bool CharacterIsDigit(char c) {
		return c >= '0' && c <= '9';
	}
	static bool CharacterIsHex(char c) {
		return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F');
	}
	static char CharacterToLower(char c) {
		if (c >= 'A' && c <= 'Z') {
			return c - ('A' - 'a');
		}
		return c;
	}
	static char CharacterIsAlpha(char c) {
		return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z');
	}
	static bool CharacterIsOperator(char c) {
		if (c == '_') {
			return false;
		}
		if (c >= '!' && c <= '/') {
			return true;
		}
		if (c >= ':' && c <= '@') {
			return true;
		}
		if (c >= '[' && c <= '`') {
			return true;
		}
		if (c >= '{' && c <= '~') {
			return true;
		}
		return false;
	}

	template <class TO>
	static vector<TO> ConvertStrings(const vector<string> &strings) {
		vector<TO> result;
		for (auto &string : strings) {
			result.emplace_back(string);
		}
		return result;
	}

	static vector<SQLIdentifier> ConvertToSQLIdentifiers(const vector<string> &strings) {
		return ConvertStrings<SQLIdentifier>(strings);
	}

	static vector<SQLString> ConvertToSQLStrings(const vector<string> &strings) {
		return ConvertStrings<SQLString>(strings);
	}

	//! Returns true if the needle string exists in the haystack
	DUCKDB_API static bool Contains(const string &haystack, const string &needle);

	//! Returns true if the target string starts with the given prefix
	DUCKDB_API static bool StartsWith(string str, string prefix);

	//! Returns true if the target string <b>ends</b> with the given suffix.
	DUCKDB_API static bool EndsWith(const string &str, const string &suffix);

	//! Repeat a string multiple times
	DUCKDB_API static string Repeat(const string &str, const idx_t n);

	//! Split the input string based on newline char
	DUCKDB_API static vector<string> Split(const string &str, char delimiter);

	//! Split the input string allong a quote. Note that any escaping is NOT supported.
	DUCKDB_API static vector<string> SplitWithQuote(const string &str, char delimiter = ',', char quote = '"');

	//! Join multiple strings into one string. Components are concatenated by the given separator
	DUCKDB_API static string Join(const vector<string> &input, const string &separator);

	template <class T>
	static string ToString(const vector<T> &input, const string &separator) {
		vector<string> input_list;
		for (auto &i : input) {
			input_list.push_back(i.ToString());
		}
		return StringUtil::Join(input_list, separator);
	}

	//! Join multiple items of container with given size, transformed to string
	//! using function, into one string using the given separator
	template <typename C, typename S, typename Func>
	static string Join(const C &input, S count, const string &separator, Func f) {
		// The result
		std::string result;

		// If the input isn't empty, append the first element. We do this so we
		// don't need to introduce an if into the loop.
		if (count > 0) {
			result += f(input[0]);
		}

		// Append the remaining input components, after the first
		for (size_t i = 1; i < count; i++) {
			result += separator + f(input[i]);
		}

		return result;
	}

	//! Return a string that formats the give number of bytes
	DUCKDB_API static string BytesToHumanReadableString(idx_t bytes);

	//! Convert a string to uppercase
	DUCKDB_API static string Upper(const string &str);

	//! Convert a string to lowercase
	DUCKDB_API static string Lower(const string &str);

	DUCKDB_API static bool IsLower(const string &str);

	//! Case insensitive hash
	DUCKDB_API static uint64_t CIHash(const string &str);

	//! Case insensitive equals
	DUCKDB_API static bool CIEquals(const string &l1, const string &l2);

	//! Format a string using printf semantics
	template <typename... Args>
	static string Format(const string fmt_str, Args... params) {
		return Exception::ConstructMessage(fmt_str, params...);
	}

	//! Split the input string into a vector of strings based on the split string
	DUCKDB_API static vector<string> Split(const string &input, const string &split);

	//! Remove the whitespace char in the left end of the string
	DUCKDB_API static void LTrim(string &str);
	//! Remove the whitespace char in the right end of the string
	DUCKDB_API static void RTrim(string &str);
	//! Remove the all chars from chars_to_trim char in the right end of the string
	DUCKDB_API static void RTrim(string &str, const string &chars_to_trim);
	//! Remove the whitespace char in the left and right end of the string
	DUCKDB_API static void Trim(string &str);

	DUCKDB_API static string Replace(string source, const string &from, const string &to);

	//! Get the levenshtein distance from two strings
	//! The not_equal_penalty is the penalty given when two characters in a string are not equal
	//! The regular levenshtein distance has a not equal penalty of 1, which means changing a character is as expensive
	//! as adding or removing one For similarity searches we often want to give extra weight to changing a character For
	//! example: with an equal penalty of 1, "pg_am" is closer to "depdelay" than "depdelay_minutes"
	//! with an equal penalty of 3, "depdelay_minutes" is closer to "depdelay" than to "pg_am"
	DUCKDB_API static idx_t LevenshteinDistance(const string &s1, const string &s2, idx_t not_equal_penalty = 1);

	//! Returns the similarity score between two strings
	DUCKDB_API static idx_t SimilarityScore(const string &s1, const string &s2);
	//! Get the top-n strings (sorted by the given score distance) from a set of scores.
	//! At least one entry is returned (if there is one).
	//! Strings are only returned if they have a score less than the threshold.
	DUCKDB_API static vector<string> TopNStrings(vector<std::pair<string, idx_t>> scores, idx_t n = 5,
	                                             idx_t threshold = 5);
	//! Computes the levenshtein distance of each string in strings, and compares it to target, then returns TopNStrings
	//! with the given params.
	DUCKDB_API static vector<string> TopNLevenshtein(const vector<string> &strings, const string &target, idx_t n = 5,
	                                                 idx_t threshold = 5);
	DUCKDB_API static string CandidatesMessage(const vector<string> &candidates,
	                                           const string &candidate = "Candidate bindings");

	//! Generate an error message in the form of "{message_prefix}: nearest_string, nearest_string2, ...
	//! Equivalent to calling TopNLevenshtein followed by CandidatesMessage
	DUCKDB_API static string CandidatesErrorMessage(const vector<string> &strings, const string &target,
	                                                const string &message_prefix, idx_t n = 5);

	//! Returns true if two null-terminated strings are equal or point to the same address.
	//! Returns false if only one of the strings is nullptr
	static bool Equals(const char *s1, const char *s2) {
		if (s1 == s2) {
			return true;
		}
		if (s1 == nullptr || s2 == nullptr) {
			return false;
		}
		return strcmp(s1, s2) == 0;
	}
};

} // namespace duckdb




#include <functional>

namespace duckdb {

struct date_t;
struct dtime_t;

//! Type used to represent timestamps (seconds,microseconds,milliseconds or nanoseconds since 1970-01-01)
struct timestamp_t { // NOLINT
	int64_t value;

	timestamp_t() = default;
	explicit inline timestamp_t(int64_t value_p) : value(value_p) {
	}
	inline timestamp_t &operator=(int64_t value_p) {
		value = value_p;
		return *this;
	}

	// explicit conversion
	explicit inline operator int64_t() const {
		return value;
	}

	// comparison operators
	inline bool operator==(const timestamp_t &rhs) const {
		return value == rhs.value;
	};
	inline bool operator!=(const timestamp_t &rhs) const {
		return value != rhs.value;
	};
	inline bool operator<=(const timestamp_t &rhs) const {
		return value <= rhs.value;
	};
	inline bool operator<(const timestamp_t &rhs) const {
		return value < rhs.value;
	};
	inline bool operator>(const timestamp_t &rhs) const {
		return value > rhs.value;
	};
	inline bool operator>=(const timestamp_t &rhs) const {
		return value >= rhs.value;
	};

	// arithmetic operators
	timestamp_t operator+(const double &value) const;
	int64_t operator-(const timestamp_t &other) const;

	// in-place operators
	timestamp_t &operator+=(const int64_t &delta);
	timestamp_t &operator-=(const int64_t &delta);

	// special values
	static timestamp_t infinity() { // NOLINT
		return timestamp_t(NumericLimits<int64_t>::Maximum());
	}                                // NOLINT
	static timestamp_t ninfinity() { // NOLINT
		return timestamp_t(-NumericLimits<int64_t>::Maximum());
	}                                   // NOLINT
	static inline timestamp_t epoch() { // NOLINT
		return timestamp_t(0);
	} // NOLINT
};

struct timestamp_tz_t : public timestamp_t {};  // NOLINT
struct timestamp_ns_t : public timestamp_t {};  // NOLINT
struct timestamp_ms_t : public timestamp_t {};  // NOLINT
struct timestamp_sec_t : public timestamp_t {}; // NOLINT

enum class TimestampCastResult : uint8_t { SUCCESS, ERROR_INCORRECT_FORMAT, ERROR_NON_UTC_TIMEZONE };

//! The Timestamp class is a static class that holds helper functions for the Timestamp
//! type.
class Timestamp {
public:
	// min timestamp is 290308-12-22 (BC)
	constexpr static const int32_t MIN_YEAR = -290308;
	constexpr static const int32_t MIN_MONTH = 12;
	constexpr static const int32_t MIN_DAY = 22;

public:
	//! Convert a string in the format "YYYY-MM-DD hh:mm:ss[.f][-+TH[:tm]]" to a timestamp object
	DUCKDB_API static timestamp_t FromString(const string &str);
	//! Convert a string where the offset can also be a time zone string: / [A_Za-z0-9/_]+/
	//! If has_offset is true, then the result is an instant that was offset from UTC
	//! If the tz is not empty, the result is still an instant, but the parts can be extracted and applied to the TZ
	DUCKDB_API static bool TryConvertTimestampTZ(const char *str, idx_t len, timestamp_t &result, bool &has_offset,
	                                             string_t &tz);
	DUCKDB_API static TimestampCastResult TryConvertTimestamp(const char *str, idx_t len, timestamp_t &result);
	DUCKDB_API static timestamp_t FromCString(const char *str, idx_t len);
	//! Convert a date object to a string in the format "YYYY-MM-DD hh:mm:ss"
	DUCKDB_API static string ToString(timestamp_t timestamp);

	DUCKDB_API static date_t GetDate(timestamp_t timestamp);

	DUCKDB_API static dtime_t GetTime(timestamp_t timestamp);
	//! Create a Timestamp object from a specified (date, time) combination
	DUCKDB_API static timestamp_t FromDatetime(date_t date, dtime_t time);
	DUCKDB_API static bool TryFromDatetime(date_t date, dtime_t time, timestamp_t &result);

	//! Is the character a valid part of a time zone name?
	static inline bool CharacterIsTimeZone(char c) {
		return StringUtil::CharacterIsAlpha(c) || StringUtil::CharacterIsDigit(c) || c == '_' || c == '/' || c == '+' ||
		       c == '-';
	}

	//! Is the timestamp finite or infinite?
	static inline bool IsFinite(timestamp_t timestamp) {
		return timestamp != timestamp_t::infinity() && timestamp != timestamp_t::ninfinity();
	}

	//! Extract the date and time from a given timestamp object
	DUCKDB_API static void Convert(timestamp_t date, date_t &out_date, dtime_t &out_time);
	//! Returns current timestamp
	DUCKDB_API static timestamp_t GetCurrentTimestamp();

	//! Convert the epoch (in sec) to a timestamp
	DUCKDB_API static timestamp_t FromEpochSeconds(int64_t ms);
	//! Convert the epoch (in ms) to a timestamp
	DUCKDB_API static timestamp_t FromEpochMs(int64_t ms);
	//! Convert the epoch (in microseconds) to a timestamp
	DUCKDB_API static timestamp_t FromEpochMicroSeconds(int64_t micros);
	//! Convert the epoch (in nanoseconds) to a timestamp
	DUCKDB_API static timestamp_t FromEpochNanoSeconds(int64_t micros);

	//! Convert the epoch (in seconds) to a timestamp
	DUCKDB_API static int64_t GetEpochSeconds(timestamp_t timestamp);
	//! Convert the epoch (in ms) to a timestamp
	DUCKDB_API static int64_t GetEpochMs(timestamp_t timestamp);
	//! Convert a timestamp to epoch (in microseconds)
	DUCKDB_API static int64_t GetEpochMicroSeconds(timestamp_t timestamp);
	//! Convert a timestamp to epoch (in nanoseconds)
	DUCKDB_API static int64_t GetEpochNanoSeconds(timestamp_t timestamp);

	DUCKDB_API static bool TryParseUTCOffset(const char *str, idx_t &pos, idx_t len, int &hour_offset,
	                                         int &minute_offset);

	DUCKDB_API static string ConversionError(const string &str);
	DUCKDB_API static string ConversionError(string_t str);
	DUCKDB_API static string UnsupportedTimezoneError(const string &str);
	DUCKDB_API static string UnsupportedTimezoneError(string_t str);
};

} // namespace duckdb

namespace std {

//! Timestamp
template <>
struct hash<duckdb::timestamp_t> {
	std::size_t operator()(const duckdb::timestamp_t &k) const {
		using std::hash;
		return hash<int64_t>()((int64_t)k);
	}
};
template <>
struct hash<duckdb::timestamp_ms_t> {
	std::size_t operator()(const duckdb::timestamp_ms_t &k) const {
		using std::hash;
		return hash<int64_t>()((int64_t)k);
	}
};
template <>
struct hash<duckdb::timestamp_ns_t> {
	std::size_t operator()(const duckdb::timestamp_ns_t &k) const {
		using std::hash;
		return hash<int64_t>()((int64_t)k);
	}
};
template <>
struct hash<duckdb::timestamp_sec_t> {
	std::size_t operator()(const duckdb::timestamp_sec_t &k) const {
		using std::hash;
		return hash<int64_t>()((int64_t)k);
	}
};
template <>
struct hash<duckdb::timestamp_tz_t> {
	std::size_t operator()(const duckdb::timestamp_tz_t &k) const {
		using std::hash;
		return hash<int64_t>()((int64_t)k);
	}
};
} // namespace std

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/types/date.hpp
//
//
//===----------------------------------------------------------------------===//






//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/types/string_type.hpp
//
//
//===----------------------------------------------------------------------===//







#include <cstring>

namespace duckdb {

struct string_t {
	friend struct StringComparisonOperators;
	friend class StringSegment;

public:
	static constexpr idx_t PREFIX_BYTES = 4 * sizeof(char);
	static constexpr idx_t INLINE_BYTES = 12 * sizeof(char);
	static constexpr idx_t HEADER_SIZE = sizeof(uint32_t) + PREFIX_BYTES;
#ifndef DUCKDB_DEBUG_NO_INLINE
	static constexpr idx_t PREFIX_LENGTH = PREFIX_BYTES;
	static constexpr idx_t INLINE_LENGTH = INLINE_BYTES;
#else
	static constexpr idx_t PREFIX_LENGTH = 0;
	static constexpr idx_t INLINE_LENGTH = 0;
#endif

	string_t() = default;
	explicit string_t(uint32_t len) {
		value.inlined.length = len;
	}
	string_t(const char *data, uint32_t len) {
		value.inlined.length = len;
		D_ASSERT(data || GetSize() == 0);
		if (IsInlined()) {
			// zero initialize the prefix first
			// this makes sure that strings with length smaller than 4 still have an equal prefix
			memset(value.inlined.inlined, 0, INLINE_BYTES);
			if (GetSize() == 0) {
				return;
			}
			// small string: inlined
			memcpy(value.inlined.inlined, data, GetSize());
		} else {
			// large string: store pointer
#ifndef DUCKDB_DEBUG_NO_INLINE
			memcpy(value.pointer.prefix, data, PREFIX_LENGTH);
#else
			memset(value.pointer.prefix, 0, PREFIX_BYTES);
#endif
			value.pointer.ptr = (char *)data; // NOLINT
		}
	}
	string_t(const char *data) : string_t(data, strlen(data)) { // NOLINT: Allow implicit conversion from `const char*`
	}
	string_t(const string &value)
	    : string_t(value.c_str(), value.size()) { // NOLINT: Allow implicit conversion from `const char*`
	}

	bool IsInlined() const {
		return GetSize() <= INLINE_LENGTH;
	}

	const char *GetData() const {
		return IsInlined() ? const_char_ptr_cast(value.inlined.inlined) : value.pointer.ptr;
	}
	const char *GetDataUnsafe() const {
		return GetData();
	}

	char *GetDataWriteable() const {
		return IsInlined() ? (char *)value.inlined.inlined : value.pointer.ptr; // NOLINT
	}

	const char *GetPrefix() const {
		return value.pointer.prefix;
	}

	idx_t GetSize() const {
		return value.inlined.length;
	}

	string GetString() const {
		return string(GetData(), GetSize());
	}

	explicit operator string() const {
		return GetString();
	}

	void SetPointer(char *new_ptr) {
		D_ASSERT(!IsInlined());
		value.pointer.ptr = new_ptr;
	}

	void Finalize() {
		// set trailing NULL byte
		if (GetSize() <= INLINE_LENGTH) {
			// fill prefix with zeros if the length is smaller than the prefix length
			for (idx_t i = GetSize(); i < INLINE_BYTES; i++) {
				value.inlined.inlined[i] = '\0';
			}
		} else {
			// copy the data into the prefix
#ifndef DUCKDB_DEBUG_NO_INLINE
			auto dataptr = GetData();
			memcpy(value.pointer.prefix, dataptr, PREFIX_LENGTH);
#else
			memset(value.pointer.prefix, 0, PREFIX_BYTES);
#endif
		}
	}

	void Verify() const;
	void VerifyNull() const;

	struct StringComparisonOperators {
		static inline bool Equals(const string_t &a, const string_t &b) {
#ifdef DUCKDB_DEBUG_NO_INLINE
			if (a.GetSize() != b.GetSize())
				return false;
			return (memcmp(a.GetData(), b.GetData(), a.GetSize()) == 0);
#endif
			uint64_t A = Load<uint64_t>(const_data_ptr_cast(&a));
			uint64_t B = Load<uint64_t>(const_data_ptr_cast(&b));
			if (A != B) {
				// Either length or prefix are different -> not equal
				return false;
			}
			// they have the same length and same prefix!
			A = Load<uint64_t>(const_data_ptr_cast(&a) + 8u);
			B = Load<uint64_t>(const_data_ptr_cast(&b) + 8u);
			if (A == B) {
				// either they are both inlined (so compare equal) or point to the same string (so compare equal)
				return true;
			}
			if (!a.IsInlined()) {
				// 'long' strings of the same length -> compare pointed value
				if (memcmp(a.value.pointer.ptr, b.value.pointer.ptr, a.GetSize()) == 0) {
					return true;
				}
			}
			// either they are short string of same length but different content
			//     or they point to string with different content
			//     either way, they can't represent the same underlying string
			return false;
		}
		// compare up to shared length. if still the same, compare lengths
		static bool GreaterThan(const string_t &left, const string_t &right) {
			const uint32_t left_length = left.GetSize();
			const uint32_t right_length = right.GetSize();
			const uint32_t min_length = std::min<uint32_t>(left_length, right_length);

#ifndef DUCKDB_DEBUG_NO_INLINE
			uint32_t A = Load<uint32_t>(const_data_ptr_cast(left.GetPrefix()));
			uint32_t B = Load<uint32_t>(const_data_ptr_cast(right.GetPrefix()));

			// Utility to move 0xa1b2c3d4 into 0xd4c3b2a1, basically inverting the order byte-a-byte
			auto bswap = [](uint32_t v) -> uint32_t {
				uint32_t t1 = (v >> 16u) | (v << 16u);
				uint32_t t2 = t1 & 0x00ff00ff;
				uint32_t t3 = t1 & 0xff00ff00;
				return (t2 << 8u) | (t3 >> 8u);
			};

			// Check on prefix -----
			// We dont' need to mask since:
			//	if the prefix is greater(after bswap), it will stay greater regardless of the extra bytes
			// 	if the prefix is smaller(after bswap), it will stay smaller regardless of the extra bytes
			//	if the prefix is equal, the extra bytes are guaranteed to be /0 for the shorter one

			if (A != B)
				return bswap(A) > bswap(B);
#endif
			auto memcmp_res = memcmp(left.GetData(), right.GetData(), min_length);
			return memcmp_res > 0 || (memcmp_res == 0 && left_length > right_length);
		}
	};

	bool operator==(const string_t &r) const {
		return StringComparisonOperators::Equals(*this, r);
	}

	bool operator>(const string_t &r) const {
		return StringComparisonOperators::GreaterThan(*this, r);
	}
	bool operator<(const string_t &r) const {
		return r > *this;
	}

private:
	union {
		struct {
			uint32_t length;
			char prefix[4];
			char *ptr;
		} pointer;
		struct {
			uint32_t length;
			char inlined[12];
		} inlined;
	} value;
};

} // namespace duckdb



#include <functional>

namespace duckdb {

struct timestamp_t;

//! Type used to represent dates (days since 1970-01-01)
struct date_t { // NOLINT
	int32_t days;

	date_t() = default;
	explicit inline date_t(int32_t days_p) : days(days_p) {
	}

	// explicit conversion
	explicit inline operator int32_t() const {
		return days;
	}

	// comparison operators
	inline bool operator==(const date_t &rhs) const {
		return days == rhs.days;
	};
	inline bool operator!=(const date_t &rhs) const {
		return days != rhs.days;
	};
	inline bool operator<=(const date_t &rhs) const {
		return days <= rhs.days;
	};
	inline bool operator<(const date_t &rhs) const {
		return days < rhs.days;
	};
	inline bool operator>(const date_t &rhs) const {
		return days > rhs.days;
	};
	inline bool operator>=(const date_t &rhs) const {
		return days >= rhs.days;
	};

	// arithmetic operators
	inline date_t operator+(const int32_t &days) const {
		return date_t(this->days + days);
	};
	inline date_t operator-(const int32_t &days) const {
		return date_t(this->days - days);
	};

	// in-place operators
	inline date_t &operator+=(const int32_t &days) {
		this->days += days;
		return *this;
	};
	inline date_t &operator-=(const int32_t &days) {
		this->days -= days;
		return *this;
	};

	// special values
	static inline date_t infinity() {
		return date_t(NumericLimits<int32_t>::Maximum());
	} // NOLINT
	static inline date_t ninfinity() {
		return date_t(-NumericLimits<int32_t>::Maximum());
	} // NOLINT
	static inline date_t epoch() {
		return date_t(0);
	} // NOLINT
};

//! The Date class is a static class that holds helper functions for the Date type.
class Date {
public:
	static const char *PINF;  // NOLINT
	static const char *NINF;  // NOLINT
	static const char *EPOCH; // NOLINT

	static const string_t MONTH_NAMES[12];
	static const string_t MONTH_NAMES_ABBREVIATED[12];
	static const string_t DAY_NAMES[7];
	static const string_t DAY_NAMES_ABBREVIATED[7];
	static const int32_t NORMAL_DAYS[13];
	static const int32_t CUMULATIVE_DAYS[13];
	static const int32_t LEAP_DAYS[13];
	static const int32_t CUMULATIVE_LEAP_DAYS[13];
	static const int32_t CUMULATIVE_YEAR_DAYS[401];
	static const int8_t MONTH_PER_DAY_OF_YEAR[365];
	static const int8_t LEAP_MONTH_PER_DAY_OF_YEAR[366];

	// min date is 5877642-06-25 (BC) (-2^31+2)
	constexpr static const int32_t DATE_MIN_YEAR = -5877641;
	constexpr static const int32_t DATE_MIN_MONTH = 6;
	constexpr static const int32_t DATE_MIN_DAY = 25;
	// max date is 5881580-07-10 (2^31-2)
	constexpr static const int32_t DATE_MAX_YEAR = 5881580;
	constexpr static const int32_t DATE_MAX_MONTH = 7;
	constexpr static const int32_t DATE_MAX_DAY = 10;
	constexpr static const int32_t EPOCH_YEAR = 1970;

	constexpr static const int32_t YEAR_INTERVAL = 400;
	constexpr static const int32_t DAYS_PER_YEAR_INTERVAL = 146097;

public:
	//! Convert a string in the format "YYYY-MM-DD" to a date object
	DUCKDB_API static date_t FromString(const string &str, bool strict = false);
	//! Convert a string in the format "YYYY-MM-DD" to a date object
	DUCKDB_API static date_t FromCString(const char *str, idx_t len, bool strict = false);
	//! Convert a date object to a string in the format "YYYY-MM-DD"
	DUCKDB_API static string ToString(date_t date);
	//! Try to convert text in a buffer to a date; returns true if parsing was successful
	//! If the date was a "special" value, the special flag will be set.
	DUCKDB_API static bool TryConvertDate(const char *buf, idx_t len, idx_t &pos, date_t &result, bool &special,
	                                      bool strict = false);

	//! Create a string "YYYY-MM-DD" from a specified (year, month, day)
	//! combination
	DUCKDB_API static string Format(int32_t year, int32_t month, int32_t day);

	//! Extract the year, month and day from a given date object
	DUCKDB_API static void Convert(date_t date, int32_t &out_year, int32_t &out_month, int32_t &out_day);
	//! Create a Date object from a specified (year, month, day) combination
	DUCKDB_API static date_t FromDate(int32_t year, int32_t month, int32_t day);
	DUCKDB_API static bool TryFromDate(int32_t year, int32_t month, int32_t day, date_t &result);

	//! Returns true if (year) is a leap year, and false otherwise
	DUCKDB_API static bool IsLeapYear(int32_t year);

	//! Returns true if the specified (year, month, day) combination is a valid
	//! date
	DUCKDB_API static bool IsValid(int32_t year, int32_t month, int32_t day);

	//! Returns true if the specified date is finite
	static inline bool IsFinite(date_t date) {
		return date != date_t::infinity() && date != date_t::ninfinity();
	}

	//! The max number of days in a month of a given year
	DUCKDB_API static int32_t MonthDays(int32_t year, int32_t month);

	//! Extract the epoch from the date (seconds since 1970-01-01)
	DUCKDB_API static int64_t Epoch(date_t date);
	//! Extract the epoch from the date (nanoseconds since 1970-01-01)
	DUCKDB_API static int64_t EpochNanoseconds(date_t date);
	//! Extract the epoch from the date (microseconds since 1970-01-01)
	DUCKDB_API static int64_t EpochMicroseconds(date_t date);
	//! Convert the epoch (seconds since 1970-01-01) to a date_t
	DUCKDB_API static date_t EpochToDate(int64_t epoch);

	//! Extract the number of days since epoch (days since 1970-01-01)
	DUCKDB_API static int32_t EpochDays(date_t date);
	//! Convert the epoch number of days to a date_t
	DUCKDB_API static date_t EpochDaysToDate(int32_t epoch);

	//! Extract year of a date entry
	DUCKDB_API static int32_t ExtractYear(date_t date);
	//! Extract year of a date entry, but optimized to first try the last year found
	DUCKDB_API static int32_t ExtractYear(date_t date, int32_t *last_year);
	DUCKDB_API static int32_t ExtractYear(timestamp_t ts, int32_t *last_year);
	//! Extract month of a date entry
	DUCKDB_API static int32_t ExtractMonth(date_t date);
	//! Extract day of a date entry
	DUCKDB_API static int32_t ExtractDay(date_t date);
	//! Extract the day of the week (1-7)
	DUCKDB_API static int32_t ExtractISODayOfTheWeek(date_t date);
	//! Extract the day of the year
	DUCKDB_API static int32_t ExtractDayOfTheYear(date_t date);
	//! Extract the ISO week number
	//! ISO weeks start on Monday and the first week of a year
	//! contains January 4 of that year.
	//! In the ISO week-numbering system, it is possible for early-January dates
	//! to be part of the 52nd or 53rd week of the previous year.
	DUCKDB_API static void ExtractISOYearWeek(date_t date, int32_t &year, int32_t &week);
	DUCKDB_API static int32_t ExtractISOWeekNumber(date_t date);
	DUCKDB_API static int32_t ExtractISOYearNumber(date_t date);
	//! Extract the week number as Python handles it.
	//! Either Monday or Sunday is the first day of the week,
	//! and any date before the first Monday/Sunday returns week 0
	//! This is a bit more consistent because week numbers in a year are always incrementing
	DUCKDB_API static int32_t ExtractWeekNumberRegular(date_t date, bool monday_first = true);
	//! Returns the date of the monday of the current week.
	DUCKDB_API static date_t GetMondayOfCurrentWeek(date_t date);

	//! Helper function to parse two digits from a string (e.g. "30" -> 30, "03" -> 3, "3" -> 3)
	DUCKDB_API static bool ParseDoubleDigit(const char *buf, idx_t len, idx_t &pos, int32_t &result);

	DUCKDB_API static string ConversionError(const string &str);
	DUCKDB_API static string ConversionError(string_t str);

private:
	static void ExtractYearOffset(int32_t &n, int32_t &year, int32_t &year_offset);
};

} // namespace duckdb

namespace std {

//! Date
template <>
struct hash<duckdb::date_t> {
	std::size_t operator()(const duckdb::date_t &k) const {
		using std::hash;
		return hash<int32_t>()((int32_t)k);
	}
};
} // namespace std





#include <functional>

namespace duckdb {

//! Type used to represent time (microseconds)
struct dtime_t { // NOLINT
	int64_t micros;

	dtime_t() = default;
	explicit inline dtime_t(int64_t micros_p) : micros(micros_p) {
	}
	inline dtime_t &operator=(int64_t micros_p) {
		micros = micros_p;
		return *this;
	}

	// explicit conversion
	explicit inline operator int64_t() const {
		return micros;
	}
	explicit inline operator double() const {
		return micros;
	}

	// comparison operators
	inline bool operator==(const dtime_t &rhs) const {
		return micros == rhs.micros;
	};
	inline bool operator!=(const dtime_t &rhs) const {
		return micros != rhs.micros;
	};
	inline bool operator<=(const dtime_t &rhs) const {
		return micros <= rhs.micros;
	};
	inline bool operator<(const dtime_t &rhs) const {
		return micros < rhs.micros;
	};
	inline bool operator>(const dtime_t &rhs) const {
		return micros > rhs.micros;
	};
	inline bool operator>=(const dtime_t &rhs) const {
		return micros >= rhs.micros;
	};

	// arithmetic operators
	inline dtime_t operator+(const int64_t &micros) const {
		return dtime_t(this->micros + micros);
	};
	inline dtime_t operator+(const double &micros) const {
		return dtime_t(this->micros + int64_t(micros));
	};
	inline dtime_t operator-(const int64_t &micros) const {
		return dtime_t(this->micros - micros);
	};
	inline dtime_t operator*(const idx_t &copies) const {
		return dtime_t(this->micros * copies);
	};
	inline dtime_t operator/(const idx_t &copies) const {
		return dtime_t(this->micros / copies);
	};
	inline int64_t operator-(const dtime_t &other) const {
		return this->micros - other.micros;
	};

	// in-place operators
	inline dtime_t &operator+=(const int64_t &micros) {
		this->micros += micros;
		return *this;
	};
	inline dtime_t &operator-=(const int64_t &micros) {
		this->micros -= micros;
		return *this;
	};
	inline dtime_t &operator+=(const dtime_t &other) {
		this->micros += other.micros;
		return *this;
	};

	// special values
	static inline dtime_t allballs() {
		return dtime_t(0);
	} // NOLINT
};

struct dtime_tz_t : public dtime_t {};

} // namespace duckdb

namespace std {

//! Time
template <>
struct hash<duckdb::dtime_t> {
	std::size_t operator()(const duckdb::dtime_t &k) const {
		using std::hash;
		return hash<int64_t>()((int64_t)k);
	}
};
template <>
struct hash<duckdb::dtime_tz_t> {
	std::size_t operator()(const duckdb::dtime_tz_t &k) const {
		using std::hash;
		return hash<int64_t>()((int64_t)k);
	}
};
} // namespace std

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/types/interval.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

struct dtime_t;
struct date_t;
struct timestamp_t;

struct interval_t {
	int32_t months;
	int32_t days;
	int64_t micros;

	inline bool operator==(const interval_t &rhs) const {
		return this->days == rhs.days && this->months == rhs.months && this->micros == rhs.micros;
	}
};

//! The Interval class is a static class that holds helper functions for the Interval
//! type.
class Interval {
public:
	static constexpr const int32_t MONTHS_PER_MILLENIUM = 12000;
	static constexpr const int32_t MONTHS_PER_CENTURY = 1200;
	static constexpr const int32_t MONTHS_PER_DECADE = 120;
	static constexpr const int32_t MONTHS_PER_YEAR = 12;
	static constexpr const int32_t MONTHS_PER_QUARTER = 3;
	static constexpr const int32_t DAYS_PER_WEEK = 7;
	//! only used for interval comparison/ordering purposes, in which case a month counts as 30 days
	static constexpr const int64_t DAYS_PER_MONTH = 30;
	static constexpr const int64_t DAYS_PER_YEAR = 365;
	static constexpr const int64_t MSECS_PER_SEC = 1000;
	static constexpr const int32_t SECS_PER_MINUTE = 60;
	static constexpr const int32_t MINS_PER_HOUR = 60;
	static constexpr const int32_t HOURS_PER_DAY = 24;
	static constexpr const int32_t SECS_PER_HOUR = SECS_PER_MINUTE * MINS_PER_HOUR;
	static constexpr const int32_t SECS_PER_DAY = SECS_PER_HOUR * HOURS_PER_DAY;
	static constexpr const int32_t SECS_PER_WEEK = SECS_PER_DAY * DAYS_PER_WEEK;

	static constexpr const int64_t MICROS_PER_MSEC = 1000;
	static constexpr const int64_t MICROS_PER_SEC = MICROS_PER_MSEC * MSECS_PER_SEC;
	static constexpr const int64_t MICROS_PER_MINUTE = MICROS_PER_SEC * SECS_PER_MINUTE;
	static constexpr const int64_t MICROS_PER_HOUR = MICROS_PER_MINUTE * MINS_PER_HOUR;
	static constexpr const int64_t MICROS_PER_DAY = MICROS_PER_HOUR * HOURS_PER_DAY;
	static constexpr const int64_t MICROS_PER_WEEK = MICROS_PER_DAY * DAYS_PER_WEEK;
	static constexpr const int64_t MICROS_PER_MONTH = MICROS_PER_DAY * DAYS_PER_MONTH;

	static constexpr const int64_t NANOS_PER_MICRO = 1000;
	static constexpr const int64_t NANOS_PER_MSEC = NANOS_PER_MICRO * MICROS_PER_MSEC;
	static constexpr const int64_t NANOS_PER_SEC = NANOS_PER_MSEC * MSECS_PER_SEC;
	static constexpr const int64_t NANOS_PER_MINUTE = NANOS_PER_SEC * SECS_PER_MINUTE;
	static constexpr const int64_t NANOS_PER_HOUR = NANOS_PER_MINUTE * MINS_PER_HOUR;
	static constexpr const int64_t NANOS_PER_DAY = NANOS_PER_HOUR * HOURS_PER_DAY;
	static constexpr const int64_t NANOS_PER_WEEK = NANOS_PER_DAY * DAYS_PER_WEEK;

public:
	//! Convert a string to an interval object
	static bool FromString(const string &str, interval_t &result);
	//! Convert a string to an interval object
	static bool FromCString(const char *str, idx_t len, interval_t &result, string *error_message, bool strict);
	//! Convert an interval object to a string
	static string ToString(const interval_t &val);

	//! Convert milliseconds to a normalised interval
	DUCKDB_API static interval_t FromMicro(int64_t micros);

	//! Get Interval in milliseconds
	static int64_t GetMilli(const interval_t &val);

	//! Get Interval in microseconds
	static int64_t GetMicro(const interval_t &val);

	//! Get Interval in Nanoseconds
	static int64_t GetNanoseconds(const interval_t &val);

	//! Returns the age between two timestamps (including 30 day months)
	static interval_t GetAge(timestamp_t timestamp_1, timestamp_t timestamp_2);

	//! Returns the exact difference between two timestamps (days and seconds)
	static interval_t GetDifference(timestamp_t timestamp_1, timestamp_t timestamp_2);

	//! Returns the inverted interval
	static interval_t Invert(interval_t interval);

	//! Add an interval to a date
	static date_t Add(date_t left, interval_t right);
	//! Add an interval to a timestamp
	static timestamp_t Add(timestamp_t left, interval_t right);
	//! Add an interval to a time. In case the time overflows or underflows, modify the date by the overflow.
	//! For example if we go from 23:00 to 02:00, we add a day to the date
	static dtime_t Add(dtime_t left, interval_t right, date_t &date);

	//! Comparison operators
	inline static bool Equals(const interval_t &left, const interval_t &right);
	inline static bool GreaterThan(const interval_t &left, const interval_t &right);
};
static void NormalizeIntervalEntries(interval_t input, int64_t &months, int64_t &days, int64_t &micros) {
	int64_t extra_months_d = input.days / Interval::DAYS_PER_MONTH;
	int64_t extra_months_micros = input.micros / Interval::MICROS_PER_MONTH;
	input.days -= extra_months_d * Interval::DAYS_PER_MONTH;
	input.micros -= extra_months_micros * Interval::MICROS_PER_MONTH;

	int64_t extra_days_micros = input.micros / Interval::MICROS_PER_DAY;
	input.micros -= extra_days_micros * Interval::MICROS_PER_DAY;

	months = input.months + extra_months_d + extra_months_micros;
	days = input.days + extra_days_micros;
	micros = input.micros;
}

bool Interval::Equals(const interval_t &left, const interval_t &right) {
	return left.months == right.months && left.days == right.days && left.micros == right.micros;
}

bool Interval::GreaterThan(const interval_t &left, const interval_t &right) {
	int64_t lmonths, ldays, lmicros;
	int64_t rmonths, rdays, rmicros;
	NormalizeIntervalEntries(left, lmonths, ldays, lmicros);
	NormalizeIntervalEntries(right, rmonths, rdays, rmicros);

	if (lmonths > rmonths) {
		return true;
	} else if (lmonths < rmonths) {
		return false;
	}
	if (ldays > rdays) {
		return true;
	} else if (ldays < rdays) {
		return false;
	}
	return lmicros > rmicros;
}

} // namespace duckdb


namespace duckdb {

class CastFunctionSet;
class Deserializer;
class Serializer;
struct GetCastFunctionInput;
struct ExtraValueInfo;

//! The Value object holds a single arbitrary value of any type that can be
//! stored in the database.
class Value {
	friend struct StringValue;
	friend struct StructValue;
	friend struct ListValue;
	friend struct UnionValue;

public:
	//! Create an empty NULL value of the specified type
	DUCKDB_API explicit Value(LogicalType type = LogicalType::SQLNULL);
	//! Create an INTEGER value
	DUCKDB_API Value(int32_t val); // NOLINT: Allow implicit conversion from `int32_t`
	//! Create a BIGINT value
	DUCKDB_API Value(int64_t val); // NOLINT: Allow implicit conversion from `int64_t`
	//! Create a FLOAT value
	DUCKDB_API Value(float val); // NOLINT: Allow implicit conversion from `float`
	//! Create a DOUBLE value
	DUCKDB_API Value(double val); // NOLINT: Allow implicit conversion from `double`
	//! Create a VARCHAR value
	DUCKDB_API Value(const char *val); // NOLINT: Allow implicit conversion from `const char *`
	//! Create a NULL value
	DUCKDB_API Value(std::nullptr_t val); // NOLINT: Allow implicit conversion from `nullptr_t`
	//! Create a VARCHAR value
	DUCKDB_API Value(string_t val); // NOLINT: Allow implicit conversion from `string_t`
	//! Create a VARCHAR value
	DUCKDB_API Value(string val); // NOLINT: Allow implicit conversion from `string`
	//! Copy constructor
	DUCKDB_API Value(const Value &other);
	//! Move constructor
	DUCKDB_API Value(Value &&other) noexcept;
	//! Destructor
	DUCKDB_API ~Value();

	// copy assignment
	DUCKDB_API Value &operator=(const Value &other);
	// move assignment
	DUCKDB_API Value &operator=(Value &&other) noexcept;

	inline LogicalType &GetTypeMutable() {
		return type_;
	}
	inline const LogicalType &type() const { // NOLINT
		return type_;
	}
	inline bool IsNull() const {
		return is_null;
	}

	//! Create the lowest possible value of a given type (numeric only)
	DUCKDB_API static Value MinimumValue(const LogicalType &type);
	//! Create the highest possible value of a given type (numeric only)
	DUCKDB_API static Value MaximumValue(const LogicalType &type);
	//! Create the negative infinite value of a given type (numeric only)
	DUCKDB_API static Value NegativeInfinity(const LogicalType &type);
	//! Create the positive infinite value of a given type (numeric only)
	DUCKDB_API static Value Infinity(const LogicalType &type);
	//! Create a Numeric value of the specified type with the specified value
	DUCKDB_API static Value Numeric(const LogicalType &type, int64_t value);
	DUCKDB_API static Value Numeric(const LogicalType &type, hugeint_t value);

	//! Create a tinyint Value from a specified value
	DUCKDB_API static Value BOOLEAN(int8_t value);
	//! Create a tinyint Value from a specified value
	DUCKDB_API static Value TINYINT(int8_t value);
	//! Create a smallint Value from a specified value
	DUCKDB_API static Value SMALLINT(int16_t value);
	//! Create an integer Value from a specified value
	DUCKDB_API static Value INTEGER(int32_t value);
	//! Create a bigint Value from a specified value
	DUCKDB_API static Value BIGINT(int64_t value);
	//! Create an unsigned tinyint Value from a specified value
	DUCKDB_API static Value UTINYINT(uint8_t value);
	//! Create an unsigned smallint Value from a specified value
	DUCKDB_API static Value USMALLINT(uint16_t value);
	//! Create an unsigned integer Value from a specified value
	DUCKDB_API static Value UINTEGER(uint32_t value);
	//! Create an unsigned bigint Value from a specified value
	DUCKDB_API static Value UBIGINT(uint64_t value);
	//! Create a hugeint Value from a specified value
	DUCKDB_API static Value HUGEINT(hugeint_t value);
	//! Create a uuid Value from a specified value
	DUCKDB_API static Value UUID(const string &value);
	//! Create a uuid Value from a specified value
	DUCKDB_API static Value UUID(hugeint_t value);
	//! Create a hash Value from a specified value
	DUCKDB_API static Value HASH(hash_t value);
	//! Create a pointer Value from a specified value
	DUCKDB_API static Value POINTER(uintptr_t value);
	//! Create a date Value from a specified date
	DUCKDB_API static Value DATE(date_t date);
	//! Create a date Value from a specified date
	DUCKDB_API static Value DATE(int32_t year, int32_t month, int32_t day);
	//! Create a time Value from a specified time
	DUCKDB_API static Value TIME(dtime_t time);
	DUCKDB_API static Value TIMETZ(dtime_t time);
	//! Create a time Value from a specified time
	DUCKDB_API static Value TIME(int32_t hour, int32_t min, int32_t sec, int32_t micros);
	//! Create a timestamp Value from a specified date/time combination
	DUCKDB_API static Value TIMESTAMP(date_t date, dtime_t time);
	//! Create a timestamp Value from a specified timestamp
	DUCKDB_API static Value TIMESTAMP(timestamp_t timestamp);
	DUCKDB_API static Value TIMESTAMPNS(timestamp_t timestamp);
	DUCKDB_API static Value TIMESTAMPMS(timestamp_t timestamp);
	DUCKDB_API static Value TIMESTAMPSEC(timestamp_t timestamp);
	DUCKDB_API static Value TIMESTAMPTZ(timestamp_t timestamp);
	//! Create a timestamp Value from a specified timestamp in separate values
	DUCKDB_API static Value TIMESTAMP(int32_t year, int32_t month, int32_t day, int32_t hour, int32_t min, int32_t sec,
	                                  int32_t micros);
	DUCKDB_API static Value INTERVAL(int32_t months, int32_t days, int64_t micros);
	DUCKDB_API static Value INTERVAL(interval_t interval);

	// Create a enum Value from a specified uint value
	DUCKDB_API static Value ENUM(uint64_t value, const LogicalType &original_type);

	// Decimal values
	DUCKDB_API static Value DECIMAL(int16_t value, uint8_t width, uint8_t scale);
	DUCKDB_API static Value DECIMAL(int32_t value, uint8_t width, uint8_t scale);
	DUCKDB_API static Value DECIMAL(int64_t value, uint8_t width, uint8_t scale);
	DUCKDB_API static Value DECIMAL(hugeint_t value, uint8_t width, uint8_t scale);
	//! Create a float Value from a specified value
	DUCKDB_API static Value FLOAT(float value);
	//! Create a double Value from a specified value
	DUCKDB_API static Value DOUBLE(double value);
	//! Create a struct value with given list of entries
	DUCKDB_API static Value STRUCT(child_list_t<Value> values);
	//! Create a list value with the given entries, list type is inferred from children
	//! Cannot be called with an empty list, use either EMPTYLIST or LIST with a type instead
	DUCKDB_API static Value LIST(vector<Value> values);
	//! Create a list value with the given entries
	DUCKDB_API static Value LIST(const LogicalType &child_type, vector<Value> values);
	//! Create an empty list with the specified child-type
	DUCKDB_API static Value EMPTYLIST(const LogicalType &child_type);
	//! Create a map value with the given entries
	DUCKDB_API static Value MAP(const LogicalType &child_type, vector<Value> values);
	//! Create a union value from a selected value and a tag from a set of alternatives.
	DUCKDB_API static Value UNION(child_list_t<LogicalType> members, uint8_t tag, Value value);

	//! Create a blob Value from a data pointer and a length: no bytes are interpreted
	DUCKDB_API static Value BLOB(const_data_ptr_t data, idx_t len);
	static Value BLOB_RAW(const string &data) { // NOLINT
		return Value::BLOB(const_data_ptr_cast(data.c_str()), data.size());
	}
	//! Creates a blob by casting a specified string to a blob (i.e. interpreting \x characters)
	DUCKDB_API static Value BLOB(const string &data);
	//! Creates a bitstring by casting a specified string to a bitstring
	DUCKDB_API static Value BIT(const_data_ptr_t data, idx_t len);
	DUCKDB_API static Value BIT(const string &data);

	template <class T>
	T GetValue() const;
	template <class T>
	static Value CreateValue(T value);
	// Returns the internal value. Unlike GetValue(), this method does not perform casting, and assumes T matches the
	// type of the value. Only use this if you know what you are doing.
	template <class T>
	T GetValueUnsafe() const;
	//! Returns a reference to the internal value. This can only be used for primitive types.
	template <class T>
	T &GetReferenceUnsafe();

	//! Return a copy of this value
	Value Copy() const {
		return Value(*this);
	}

	//! Hashes the Value
	DUCKDB_API hash_t Hash() const;
	//! Convert this value to a string
	DUCKDB_API string ToString() const;
	//! Convert this value to a SQL-parseable string
	DUCKDB_API string ToSQLString() const;

	DUCKDB_API uintptr_t GetPointer() const;

	//! Cast this value to another type, throws exception if its not possible
	DUCKDB_API Value CastAs(CastFunctionSet &set, GetCastFunctionInput &get_input, const LogicalType &target_type,
	                        bool strict = false) const;
	DUCKDB_API Value CastAs(ClientContext &context, const LogicalType &target_type, bool strict = false) const;
	DUCKDB_API Value DefaultCastAs(const LogicalType &target_type, bool strict = false) const;
	//! Tries to cast this value to another type, and stores the result in "new_value"
	DUCKDB_API bool TryCastAs(CastFunctionSet &set, GetCastFunctionInput &get_input, const LogicalType &target_type,
	                          Value &new_value, string *error_message, bool strict = false) const;
	DUCKDB_API bool TryCastAs(ClientContext &context, const LogicalType &target_type, Value &new_value,
	                          string *error_message, bool strict = false) const;
	DUCKDB_API bool DefaultTryCastAs(const LogicalType &target_type, Value &new_value, string *error_message,
	                                 bool strict = false) const;
	//! Tries to cast this value to another type, and stores the result in THIS value again
	DUCKDB_API bool TryCastAs(CastFunctionSet &set, GetCastFunctionInput &get_input, const LogicalType &target_type,
	                          bool strict = false);
	DUCKDB_API bool TryCastAs(ClientContext &context, const LogicalType &target_type, bool strict = false);
	DUCKDB_API bool DefaultTryCastAs(const LogicalType &target_type, bool strict = false);

	DUCKDB_API void Reinterpret(LogicalType new_type);

	//! Serializes a Value to a stand-alone binary blob
	DUCKDB_API void Serialize(Serializer &serializer) const;
	DUCKDB_API void FormatSerialize(FormatSerializer &serializer) const;
	DUCKDB_API static Value FormatDeserialize(FormatDeserializer &deserializer);
	//! Deserializes a Value from a blob
	DUCKDB_API static Value Deserialize(Deserializer &source);

	//===--------------------------------------------------------------------===//
	// Comparison Operators
	//===--------------------------------------------------------------------===//
	DUCKDB_API bool operator==(const Value &rhs) const;
	DUCKDB_API bool operator!=(const Value &rhs) const;
	DUCKDB_API bool operator<(const Value &rhs) const;
	DUCKDB_API bool operator>(const Value &rhs) const;
	DUCKDB_API bool operator<=(const Value &rhs) const;
	DUCKDB_API bool operator>=(const Value &rhs) const;

	DUCKDB_API bool operator==(const int64_t &rhs) const;
	DUCKDB_API bool operator!=(const int64_t &rhs) const;
	DUCKDB_API bool operator<(const int64_t &rhs) const;
	DUCKDB_API bool operator>(const int64_t &rhs) const;
	DUCKDB_API bool operator<=(const int64_t &rhs) const;
	DUCKDB_API bool operator>=(const int64_t &rhs) const;

	DUCKDB_API static bool FloatIsFinite(float value);
	DUCKDB_API static bool DoubleIsFinite(double value);
	template <class T>
	static bool IsNan(T value) {
		throw InternalException("Unimplemented template type for Value::IsNan");
	}
	template <class T>
	static bool IsFinite(T value) {
		return true;
	}
	DUCKDB_API static bool StringIsValid(const char *str, idx_t length);
	static bool StringIsValid(const string &str) {
		return StringIsValid(str.c_str(), str.size());
	}

	//! Returns true if the values are (approximately) equivalent. Note this is NOT the SQL equivalence. For this
	//! function, NULL values are equivalent and floating point values that are close are equivalent.
	DUCKDB_API static bool ValuesAreEqual(CastFunctionSet &set, GetCastFunctionInput &get_input,
	                                      const Value &result_value, const Value &value);
	DUCKDB_API static bool ValuesAreEqual(ClientContext &context, const Value &result_value, const Value &value);
	DUCKDB_API static bool DefaultValuesAreEqual(const Value &result_value, const Value &value);
	//! Returns true if the values are not distinct from each other, following SQL semantics for NOT DISTINCT FROM.
	DUCKDB_API static bool NotDistinctFrom(const Value &lvalue, const Value &rvalue);

	friend std::ostream &operator<<(std::ostream &out, const Value &val) {
		out << val.ToString();
		return out;
	}
	DUCKDB_API void Print() const;

private:
	//! The logical of the value
	LogicalType type_; // NOLINT

	//! Whether or not the value is NULL
	bool is_null;

	//! The value of the object, if it is of a constant size Type
	union Val {
		int8_t boolean;
		int8_t tinyint;
		int16_t smallint;
		int32_t integer;
		int64_t bigint;
		uint8_t utinyint;
		uint16_t usmallint;
		uint32_t uinteger;
		uint64_t ubigint;
		hugeint_t hugeint;
		float float_;   // NOLINT
		double double_; // NOLINT
		uintptr_t pointer;
		uint64_t hash;
		date_t date;
		dtime_t time;
		timestamp_t timestamp;
		interval_t interval;
	} value_; // NOLINT

	shared_ptr<ExtraValueInfo> value_info_; // NOLINT

private:
	template <class T>
	T GetValueInternal() const;
};

//===--------------------------------------------------------------------===//
// Type-specific getters
//===--------------------------------------------------------------------===//
// Note that these are equivalent to calling GetValueUnsafe<X>, meaning no cast will be performed
// instead, an assertion will be triggered if the value is not of the correct type
struct BooleanValue {
	DUCKDB_API static bool Get(const Value &value);
};

struct TinyIntValue {
	DUCKDB_API static int8_t Get(const Value &value);
};

struct SmallIntValue {
	DUCKDB_API static int16_t Get(const Value &value);
};

struct IntegerValue {
	DUCKDB_API static int32_t Get(const Value &value);
};

struct BigIntValue {
	DUCKDB_API static int64_t Get(const Value &value);
};

struct HugeIntValue {
	DUCKDB_API static hugeint_t Get(const Value &value);
};

struct UTinyIntValue {
	DUCKDB_API static uint8_t Get(const Value &value);
};

struct USmallIntValue {
	DUCKDB_API static uint16_t Get(const Value &value);
};

struct UIntegerValue {
	DUCKDB_API static uint32_t Get(const Value &value);
};

struct UBigIntValue {
	DUCKDB_API static uint64_t Get(const Value &value);
};

struct FloatValue {
	DUCKDB_API static float Get(const Value &value);
};

struct DoubleValue {
	DUCKDB_API static double Get(const Value &value);
};

struct StringValue {
	DUCKDB_API static const string &Get(const Value &value);
};

struct DateValue {
	DUCKDB_API static date_t Get(const Value &value);
};

struct TimeValue {
	DUCKDB_API static dtime_t Get(const Value &value);
};

struct TimestampValue {
	DUCKDB_API static timestamp_t Get(const Value &value);
};

struct IntervalValue {
	DUCKDB_API static interval_t Get(const Value &value);
};

struct StructValue {
	DUCKDB_API static const vector<Value> &GetChildren(const Value &value);
};

struct ListValue {
	DUCKDB_API static const vector<Value> &GetChildren(const Value &value);
};

struct UnionValue {
	DUCKDB_API static const Value &GetValue(const Value &value);
	DUCKDB_API static uint8_t GetTag(const Value &value);
};

//! Return the internal integral value for any type that is stored as an integral value internally
//! This can be used on values of type integer, uinteger, but also date, timestamp, decimal, etc
struct IntegralValue {
	static hugeint_t Get(const Value &value);
};

template <>
Value DUCKDB_API Value::CreateValue(bool value);
template <>
Value DUCKDB_API Value::CreateValue(uint8_t value);
template <>
Value DUCKDB_API Value::CreateValue(uint16_t value);
template <>
Value DUCKDB_API Value::CreateValue(uint32_t value);
template <>
Value DUCKDB_API Value::CreateValue(uint64_t value);
template <>
Value DUCKDB_API Value::CreateValue(int8_t value);
template <>
Value DUCKDB_API Value::CreateValue(int16_t value);
template <>
Value DUCKDB_API Value::CreateValue(int32_t value);
template <>
Value DUCKDB_API Value::CreateValue(int64_t value);
template <>
Value DUCKDB_API Value::CreateValue(hugeint_t value);
template <>
Value DUCKDB_API Value::CreateValue(date_t value);
template <>
Value DUCKDB_API Value::CreateValue(dtime_t value);
template <>
Value DUCKDB_API Value::CreateValue(dtime_tz_t value);
template <>
Value DUCKDB_API Value::CreateValue(timestamp_t value);
template <>
Value DUCKDB_API Value::CreateValue(timestamp_sec_t value);
template <>
Value DUCKDB_API Value::CreateValue(timestamp_ms_t value);
template <>
Value DUCKDB_API Value::CreateValue(timestamp_ns_t value);
template <>
Value DUCKDB_API Value::CreateValue(timestamp_tz_t value);
template <>
Value DUCKDB_API Value::CreateValue(const char *value);
template <>
Value DUCKDB_API Value::CreateValue(string value);
template <>
Value DUCKDB_API Value::CreateValue(string_t value);
template <>
Value DUCKDB_API Value::CreateValue(float value);
template <>
Value DUCKDB_API Value::CreateValue(double value);
template <>
Value DUCKDB_API Value::CreateValue(interval_t value);
template <>
Value DUCKDB_API Value::CreateValue(Value value);

template <>
DUCKDB_API bool Value::GetValue() const;
template <>
DUCKDB_API int8_t Value::GetValue() const;
template <>
DUCKDB_API int16_t Value::GetValue() const;
template <>
DUCKDB_API int32_t Value::GetValue() const;
template <>
DUCKDB_API int64_t Value::GetValue() const;
template <>
DUCKDB_API uint8_t Value::GetValue() const;
template <>
DUCKDB_API uint16_t Value::GetValue() const;
template <>
DUCKDB_API uint32_t Value::GetValue() const;
template <>
DUCKDB_API uint64_t Value::GetValue() const;
template <>
DUCKDB_API hugeint_t Value::GetValue() const;
template <>
DUCKDB_API string Value::GetValue() const;
template <>
DUCKDB_API float Value::GetValue() const;
template <>
DUCKDB_API double Value::GetValue() const;
template <>
DUCKDB_API date_t Value::GetValue() const;
template <>
DUCKDB_API dtime_t Value::GetValue() const;
template <>
DUCKDB_API timestamp_t Value::GetValue() const;
template <>
DUCKDB_API interval_t Value::GetValue() const;
template <>
DUCKDB_API Value Value::GetValue() const;

template <>
DUCKDB_API bool Value::GetValueUnsafe() const;
template <>
DUCKDB_API int8_t Value::GetValueUnsafe() const;
template <>
DUCKDB_API int16_t Value::GetValueUnsafe() const;
template <>
DUCKDB_API int32_t Value::GetValueUnsafe() const;
template <>
DUCKDB_API int64_t Value::GetValueUnsafe() const;
template <>
DUCKDB_API hugeint_t Value::GetValueUnsafe() const;
template <>
DUCKDB_API uint8_t Value::GetValueUnsafe() const;
template <>
DUCKDB_API uint16_t Value::GetValueUnsafe() const;
template <>
DUCKDB_API uint32_t Value::GetValueUnsafe() const;
template <>
DUCKDB_API uint64_t Value::GetValueUnsafe() const;
template <>
DUCKDB_API string Value::GetValueUnsafe() const;
template <>
DUCKDB_API string_t Value::GetValueUnsafe() const;
template <>
DUCKDB_API float Value::GetValueUnsafe() const;
template <>
DUCKDB_API double Value::GetValueUnsafe() const;
template <>
DUCKDB_API date_t Value::GetValueUnsafe() const;
template <>
DUCKDB_API dtime_t Value::GetValueUnsafe() const;
template <>
DUCKDB_API timestamp_t Value::GetValueUnsafe() const;
template <>
DUCKDB_API interval_t Value::GetValueUnsafe() const;

template <>
DUCKDB_API bool Value::IsNan(float input);
template <>
DUCKDB_API bool Value::IsNan(double input);

template <>
DUCKDB_API bool Value::IsFinite(float input);
template <>
DUCKDB_API bool Value::IsFinite(double input);
template <>
DUCKDB_API bool Value::IsFinite(date_t input);
template <>
DUCKDB_API bool Value::IsFinite(timestamp_t input);

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/types/vector_buffer.hpp
//
//
//===----------------------------------------------------------------------===//





//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/types/string_heap.hpp
//
//
//===----------------------------------------------------------------------===//





//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/storage/arena_allocator.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/allocator.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {
class Allocator;
class AttachedDatabase;
class ClientContext;
class DatabaseInstance;
class ExecutionContext;
class ThreadContext;

struct AllocatorDebugInfo;

struct PrivateAllocatorData {
	PrivateAllocatorData();
	virtual ~PrivateAllocatorData();

	unique_ptr<AllocatorDebugInfo> debug_info;

	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}
	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
};

typedef data_ptr_t (*allocate_function_ptr_t)(PrivateAllocatorData *private_data, idx_t size);
typedef void (*free_function_ptr_t)(PrivateAllocatorData *private_data, data_ptr_t pointer, idx_t size);
typedef data_ptr_t (*reallocate_function_ptr_t)(PrivateAllocatorData *private_data, data_ptr_t pointer, idx_t old_size,
                                                idx_t size);

class AllocatedData {
public:
	DUCKDB_API AllocatedData();
	DUCKDB_API AllocatedData(Allocator &allocator, data_ptr_t pointer, idx_t allocated_size);
	DUCKDB_API ~AllocatedData();
	// disable copy constructors
	AllocatedData(const AllocatedData &other) = delete;
	AllocatedData &operator=(const AllocatedData &) = delete;
	//! enable move constructors
	DUCKDB_API AllocatedData(AllocatedData &&other) noexcept;
	DUCKDB_API AllocatedData &operator=(AllocatedData &&) noexcept;

	data_ptr_t get() {
		return pointer;
	}
	const_data_ptr_t get() const {
		return pointer;
	}
	idx_t GetSize() const {
		return allocated_size;
	}
	bool IsSet() {
		return pointer;
	}
	void Reset();

private:
	optional_ptr<Allocator> allocator;
	data_ptr_t pointer;
	idx_t allocated_size;
};

class Allocator {
	// 281TB ought to be enough for anybody
	static constexpr const idx_t MAXIMUM_ALLOC_SIZE = 281474976710656ULL;

public:
	DUCKDB_API Allocator();
	DUCKDB_API Allocator(allocate_function_ptr_t allocate_function_p, free_function_ptr_t free_function_p,
	                     reallocate_function_ptr_t reallocate_function_p,
	                     unique_ptr<PrivateAllocatorData> private_data);
	Allocator &operator=(Allocator &&allocator) noexcept = delete;
	DUCKDB_API ~Allocator();

	DUCKDB_API data_ptr_t AllocateData(idx_t size);
	DUCKDB_API void FreeData(data_ptr_t pointer, idx_t size);
	DUCKDB_API data_ptr_t ReallocateData(data_ptr_t pointer, idx_t old_size, idx_t new_size);

	AllocatedData Allocate(idx_t size) {
		return AllocatedData(*this, AllocateData(size), size);
	}
	static data_ptr_t DefaultAllocate(PrivateAllocatorData *private_data, idx_t size) {
		return data_ptr_cast(malloc(size));
	}
	static void DefaultFree(PrivateAllocatorData *private_data, data_ptr_t pointer, idx_t size) {
		free(pointer);
	}
	static data_ptr_t DefaultReallocate(PrivateAllocatorData *private_data, data_ptr_t pointer, idx_t old_size,
	                                    idx_t size) {
		return data_ptr_cast(realloc(pointer, size));
	}
	static Allocator &Get(ClientContext &context);
	static Allocator &Get(DatabaseInstance &db);
	static Allocator &Get(AttachedDatabase &db);

	PrivateAllocatorData *GetPrivateData() {
		return private_data.get();
	}

	DUCKDB_API static Allocator &DefaultAllocator();
	DUCKDB_API static shared_ptr<Allocator> &DefaultAllocatorReference();

private:
	allocate_function_ptr_t allocate_function;
	free_function_ptr_t free_function;
	reallocate_function_ptr_t reallocate_function;

	unique_ptr<PrivateAllocatorData> private_data;
};

template <class T>
T *AllocateArray(idx_t size) {
	return (T *)Allocator::DefaultAllocator().AllocateData(size * sizeof(T));
}

template <class T>
void DeleteArray(T *ptr, idx_t size) {
	Allocator::DefaultAllocator().FreeData(data_ptr_cast(ptr), size * sizeof(T));
}

template <typename T, typename... ARGS>
T *AllocateObject(ARGS &&... args) {
	auto data = Allocator::DefaultAllocator().AllocateData(sizeof(T));
	return new (data) T(std::forward<ARGS>(args)...);
}

template <typename T>
void DestroyObject(T *ptr) {
	ptr->~T();
	Allocator::DefaultAllocator().FreeData(data_ptr_cast(ptr), sizeof(T));
}

//! The BufferAllocator is a wrapper around the global allocator class that sends any allocations made through the
//! buffer manager. This makes the buffer manager aware of the memory usage, allowing it to potentially free
//! other blocks to make space in memory.
//! Note that there is a cost to doing so (several atomic operations will be performed on allocation/free).
//! As such this class should be used primarily for larger allocations.
struct BufferAllocator {
	DUCKDB_API static Allocator &Get(ClientContext &context);
	DUCKDB_API static Allocator &Get(DatabaseInstance &db);
	DUCKDB_API static Allocator &Get(AttachedDatabase &db);
};

} // namespace duckdb



namespace duckdb {

struct ArenaChunk {
	ArenaChunk(Allocator &allocator, idx_t size);
	~ArenaChunk();

	AllocatedData data;
	idx_t current_position;
	idx_t maximum_size;
	unsafe_unique_ptr<ArenaChunk> next;
	ArenaChunk *prev;
};

class ArenaAllocator {
	static constexpr const idx_t ARENA_ALLOCATOR_INITIAL_CAPACITY = 2048;

public:
	DUCKDB_API ArenaAllocator(Allocator &allocator, idx_t initial_capacity = ARENA_ALLOCATOR_INITIAL_CAPACITY);
	DUCKDB_API ~ArenaAllocator();

	DUCKDB_API data_ptr_t Allocate(idx_t size);
	DUCKDB_API data_ptr_t Reallocate(data_ptr_t pointer, idx_t old_size, idx_t size);

	DUCKDB_API data_ptr_t AllocateAligned(idx_t size);
	DUCKDB_API data_ptr_t ReallocateAligned(data_ptr_t pointer, idx_t old_size, idx_t size);

	//! Resets the current head and destroys all previous arena chunks
	DUCKDB_API void Reset();
	DUCKDB_API void Destroy();
	DUCKDB_API void Move(ArenaAllocator &allocator);

	DUCKDB_API ArenaChunk *GetHead();
	DUCKDB_API ArenaChunk *GetTail();

	DUCKDB_API bool IsEmpty();

	//! Returns an "Allocator" wrapper for this arena allocator
	Allocator &GetAllocator() {
		return arena_allocator;
	}

private:
	//! Internal allocator that is used by the arena allocator
	Allocator &allocator;
	idx_t current_capacity;
	unsafe_unique_ptr<ArenaChunk> head;
	ArenaChunk *tail;
	//! An allocator wrapper using this arena allocator
	Allocator arena_allocator;
};

} // namespace duckdb


namespace duckdb {
//! A string heap is the owner of a set of strings, strings can be inserted into
//! it On every insert, a pointer to the inserted string is returned The
//! returned pointer will remain valid until the StringHeap is destroyed
class StringHeap {
public:
	DUCKDB_API StringHeap(Allocator &allocator = Allocator::DefaultAllocator());

	DUCKDB_API void Destroy();
	DUCKDB_API void Move(StringHeap &other);

	//! Add a string to the string heap, returns a pointer to the string
	DUCKDB_API string_t AddString(const char *data, idx_t len);
	//! Add a string to the string heap, returns a pointer to the string
	DUCKDB_API string_t AddString(const char *data);
	//! Add a string to the string heap, returns a pointer to the string
	DUCKDB_API string_t AddString(const string &data);
	//! Add a string to the string heap, returns a pointer to the string
	DUCKDB_API string_t AddString(const string_t &data);
	//! Add a blob to the string heap; blobs can be non-valid UTF8
	DUCKDB_API string_t AddBlob(const string_t &data);
	//! Add a blob to the string heap; blobs can be non-valid UTF8
	DUCKDB_API string_t AddBlob(const char *data, idx_t len);
	//! Allocates space for an empty string of size "len" on the heap
	DUCKDB_API string_t EmptyString(idx_t len);

private:
	ArenaAllocator allocator;
};

} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/storage/buffer/buffer_handle.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/storage/storage_info.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {
class Serializer;
class Deserializer;
struct FileHandle;

#define STANDARD_ROW_GROUPS_SIZE 122880
#if STANDARD_ROW_GROUPS_SIZE < STANDARD_VECTOR_SIZE
#error Row groups should be able to hold at least one vector
#endif

#if ((STANDARD_ROW_GROUPS_SIZE % STANDARD_VECTOR_SIZE) != 0)
#error Row group size should be cleanly divisible by vector size
#endif

//! The version number of the database storage format
extern const uint64_t VERSION_NUMBER;

const char *GetDuckDBVersion(idx_t version_number);

using block_id_t = int64_t;

#define INVALID_BLOCK (-1)

// maximum block id, 2^62
#define MAXIMUM_BLOCK 4611686018427388000LL

//! The MainHeader is the first header in the storage file. The MainHeader is typically written only once for a database
//! file.
struct MainHeader {
	static constexpr idx_t MAGIC_BYTE_SIZE = 4;
	static constexpr idx_t MAGIC_BYTE_OFFSET = Storage::BLOCK_HEADER_SIZE;
	static constexpr idx_t FLAG_COUNT = 4;
	// the magic bytes in front of the file
	// should be "DUCK"
	static const char MAGIC_BYTES[];
	//! The version of the database
	uint64_t version_number;
	//! The set of flags used by the database
	uint64_t flags[FLAG_COUNT];

	static void CheckMagicBytes(FileHandle &handle);

	void Serialize(Serializer &ser);
	static MainHeader Deserialize(Deserializer &source);
};

//! The DatabaseHeader contains information about the current state of the database. Every storage file has two
//! DatabaseHeaders. On startup, the DatabaseHeader with the highest iteration count is used as the active header. When
//! a checkpoint is performed, the active DatabaseHeader is switched by increasing the iteration count of the
//! DatabaseHeader.
struct DatabaseHeader {
	//! The iteration count, increases by 1 every time the storage is checkpointed.
	uint64_t iteration;
	//! A pointer to the initial meta block
	block_id_t meta_block;
	//! A pointer to the block containing the free list
	block_id_t free_list;
	//! The number of blocks that is in the file as of this database header. If the file is larger than BLOCK_SIZE *
	//! block_count any blocks appearing AFTER block_count are implicitly part of the free_list.
	uint64_t block_count;

	void Serialize(Serializer &ser);
	static DatabaseHeader Deserialize(Deserializer &source);
};

} // namespace duckdb



namespace duckdb {
class BlockHandle;
class FileBuffer;

class BufferHandle {
public:
	DUCKDB_API BufferHandle();
	DUCKDB_API BufferHandle(shared_ptr<BlockHandle> handle, FileBuffer *node);
	DUCKDB_API ~BufferHandle();
	// disable copy constructors
	BufferHandle(const BufferHandle &other) = delete;
	BufferHandle &operator=(const BufferHandle &) = delete;
	//! enable move constructors
	DUCKDB_API BufferHandle(BufferHandle &&other) noexcept;
	DUCKDB_API BufferHandle &operator=(BufferHandle &&) noexcept;

public:
	//! Returns whether or not the BufferHandle is valid.
	DUCKDB_API bool IsValid() const;
	//! Returns a pointer to the buffer data. Handle must be valid.
	inline data_ptr_t Ptr() const {
		D_ASSERT(IsValid());
		return node->buffer;
	}
	//! Returns a pointer to the buffer data. Handle must be valid.
	inline data_ptr_t Ptr() {
		D_ASSERT(IsValid());
		return node->buffer;
	}
	//! Gets the underlying file buffer. Handle must be valid.
	DUCKDB_API FileBuffer &GetFileBuffer();
	//! Destroys the buffer handle
	DUCKDB_API void Destroy();

	const shared_ptr<BlockHandle> &GetBlockHandle() const {
		return handle;
	}

private:
	//! The block handle
	shared_ptr<BlockHandle> handle;
	//! The managed buffer node
	FileBuffer *node;
};

} // namespace duckdb


namespace duckdb {

class BufferHandle;
class VectorBuffer;
class Vector;

enum class VectorBufferType : uint8_t {
	STANDARD_BUFFER,     // standard buffer, holds a single array of data
	DICTIONARY_BUFFER,   // dictionary buffer, holds a selection vector
	VECTOR_CHILD_BUFFER, // vector child buffer: holds another vector
	STRING_BUFFER,       // string buffer, holds a string heap
	FSST_BUFFER,         // fsst compressed string buffer, holds a string heap, fsst symbol table and a string count
	STRUCT_BUFFER,       // struct buffer, holds a ordered mapping from name to child vector
	LIST_BUFFER,         // list buffer, holds a single flatvector child
	MANAGED_BUFFER,      // managed buffer, holds a buffer managed by the buffermanager
	OPAQUE_BUFFER        // opaque buffer, can be created for example by the parquet reader
};

enum class VectorAuxiliaryDataType : uint8_t {
	ARROW_AUXILIARY // Holds Arrow Chunks that this vector depends on
};

struct VectorAuxiliaryData {
	explicit VectorAuxiliaryData(VectorAuxiliaryDataType type_p)
	    : type(type_p) {

	      };
	VectorAuxiliaryDataType type;

	virtual ~VectorAuxiliaryData() {
	}

public:
	template <class TARGET>
	TARGET &Cast() {
		if (type != TARGET::TYPE) {
			throw InternalException("Failed to cast vector auxiliary data to type - type mismatch");
		}
		return reinterpret_cast<TARGET &>(*this);
	}

	template <class TARGET>
	const TARGET &Cast() const {
		if (type != TARGET::TYPE) {
			throw InternalException("Failed to cast vector auxiliary data to type - type mismatch");
		}
		return reinterpret_cast<const TARGET &>(*this);
	}
};

//! The VectorBuffer is a class used by the vector to hold its data
class VectorBuffer {
public:
	explicit VectorBuffer(VectorBufferType type) : buffer_type(type) {
	}
	explicit VectorBuffer(idx_t data_size) : buffer_type(VectorBufferType::STANDARD_BUFFER) {
		if (data_size > 0) {
			data = make_unsafe_uniq_array<data_t>(data_size);
		}
	}
	explicit VectorBuffer(unsafe_unique_array<data_t> data_p)
	    : buffer_type(VectorBufferType::STANDARD_BUFFER), data(std::move(data_p)) {
	}
	virtual ~VectorBuffer() {
	}
	VectorBuffer() {
	}

public:
	data_ptr_t GetData() {
		return data.get();
	}

	void SetData(unsafe_unique_array<data_t> new_data) {
		data = std::move(new_data);
	}

	VectorAuxiliaryData *GetAuxiliaryData() {
		return aux_data.get();
	}

	void SetAuxiliaryData(unique_ptr<VectorAuxiliaryData> aux_data_p) {
		aux_data = std::move(aux_data_p);
	}

	void MoveAuxiliaryData(VectorBuffer &source_buffer) {
		SetAuxiliaryData(std::move(source_buffer.aux_data));
	}

	static buffer_ptr<VectorBuffer> CreateStandardVector(PhysicalType type, idx_t capacity = STANDARD_VECTOR_SIZE);
	static buffer_ptr<VectorBuffer> CreateConstantVector(PhysicalType type);
	static buffer_ptr<VectorBuffer> CreateConstantVector(const LogicalType &logical_type);
	static buffer_ptr<VectorBuffer> CreateStandardVector(const LogicalType &logical_type,
	                                                     idx_t capacity = STANDARD_VECTOR_SIZE);

	inline VectorBufferType GetBufferType() const {
		return buffer_type;
	}

	inline VectorAuxiliaryDataType GetAuxiliaryDataType() const {
		return aux_data->type;
	}

protected:
	VectorBufferType buffer_type;
	unique_ptr<VectorAuxiliaryData> aux_data;
	unsafe_unique_array<data_t> data;

public:
	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}
	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
};

//! The DictionaryBuffer holds a selection vector
class DictionaryBuffer : public VectorBuffer {
public:
	explicit DictionaryBuffer(const SelectionVector &sel)
	    : VectorBuffer(VectorBufferType::DICTIONARY_BUFFER), sel_vector(sel) {
	}
	explicit DictionaryBuffer(buffer_ptr<SelectionData> data)
	    : VectorBuffer(VectorBufferType::DICTIONARY_BUFFER), sel_vector(std::move(data)) {
	}
	explicit DictionaryBuffer(idx_t count = STANDARD_VECTOR_SIZE)
	    : VectorBuffer(VectorBufferType::DICTIONARY_BUFFER), sel_vector(count) {
	}

public:
	const SelectionVector &GetSelVector() const {
		return sel_vector;
	}
	SelectionVector &GetSelVector() {
		return sel_vector;
	}
	void SetSelVector(const SelectionVector &vector) {
		this->sel_vector.Initialize(vector);
	}

private:
	SelectionVector sel_vector;
};

class VectorStringBuffer : public VectorBuffer {
public:
	VectorStringBuffer();
	explicit VectorStringBuffer(VectorBufferType type);

public:
	string_t AddString(const char *data, idx_t len) {
		return heap.AddString(data, len);
	}
	string_t AddString(string_t data) {
		return heap.AddString(data);
	}
	string_t AddBlob(string_t data) {
		return heap.AddBlob(data.GetData(), data.GetSize());
	}
	string_t EmptyString(idx_t len) {
		return heap.EmptyString(len);
	}

	void AddHeapReference(buffer_ptr<VectorBuffer> heap) {
		references.push_back(std::move(heap));
	}

private:
	//! The string heap of this buffer
	StringHeap heap;
	// References to additional vector buffers referenced by this string buffer
	vector<buffer_ptr<VectorBuffer>> references;
};

class VectorFSSTStringBuffer : public VectorStringBuffer {
public:
	VectorFSSTStringBuffer();

public:
	void AddDecoder(buffer_ptr<void> &duckdb_fsst_decoder_p) {
		duckdb_fsst_decoder = duckdb_fsst_decoder_p;
	}
	void *GetDecoder() {
		return duckdb_fsst_decoder.get();
	}
	void SetCount(idx_t count) {
		total_string_count = count;
	}
	idx_t GetCount() {
		return total_string_count;
	}

private:
	buffer_ptr<void> duckdb_fsst_decoder;
	idx_t total_string_count = 0;
};

class VectorStructBuffer : public VectorBuffer {
public:
	VectorStructBuffer();
	explicit VectorStructBuffer(const LogicalType &struct_type, idx_t capacity = STANDARD_VECTOR_SIZE);
	VectorStructBuffer(Vector &other, const SelectionVector &sel, idx_t count);
	~VectorStructBuffer() override;

public:
	const vector<unique_ptr<Vector>> &GetChildren() const {
		return children;
	}
	vector<unique_ptr<Vector>> &GetChildren() {
		return children;
	}

private:
	//! child vectors used for nested data
	vector<unique_ptr<Vector>> children;
};

class VectorListBuffer : public VectorBuffer {
public:
	explicit VectorListBuffer(unique_ptr<Vector> vector, idx_t initial_capacity = STANDARD_VECTOR_SIZE);
	explicit VectorListBuffer(const LogicalType &list_type, idx_t initial_capacity = STANDARD_VECTOR_SIZE);
	~VectorListBuffer() override;

public:
	Vector &GetChild() {
		return *child;
	}
	void Reserve(idx_t to_reserve);

	void Append(const Vector &to_append, idx_t to_append_size, idx_t source_offset = 0);
	void Append(const Vector &to_append, const SelectionVector &sel, idx_t to_append_size, idx_t source_offset = 0);

	void PushBack(const Value &insert);

	idx_t GetSize() {
		return size;
	}

	idx_t GetCapacity() {
		return capacity;
	}

	void SetCapacity(idx_t new_capacity);
	void SetSize(idx_t new_size);

private:
	//! child vectors used for nested data
	unique_ptr<Vector> child;
	idx_t capacity = 0;
	idx_t size = 0;
};

//! The ManagedVectorBuffer holds a buffer handle
class ManagedVectorBuffer : public VectorBuffer {
public:
	explicit ManagedVectorBuffer(BufferHandle handle);
	~ManagedVectorBuffer() override;

private:
	BufferHandle handle;
};

} // namespace duckdb



namespace duckdb {

struct UnifiedVectorFormat {
	const SelectionVector *sel;
	data_ptr_t data;
	ValidityMask validity;
	SelectionVector owned_sel;

	template <class T>
	static inline const T *GetData(const UnifiedVectorFormat &format) {
		return reinterpret_cast<const T *>(format.data);
	}
	template <class T>
	static inline T *GetDataNoConst(UnifiedVectorFormat &format) {
		return reinterpret_cast<T *>(format.data);
	}
};

class VectorCache;
class VectorStructBuffer;
class VectorListBuffer;

struct SelCache;

struct ConsecutiveChildListInfo {
	ConsecutiveChildListInfo() : is_constant(true), needs_slicing(false), child_list_info(list_entry_t(0, 0)) {
	}
	bool is_constant;
	bool needs_slicing;
	list_entry_t child_list_info;
};

//!  Vector of values of a specified PhysicalType.
class Vector {
	friend struct ConstantVector;
	friend struct DictionaryVector;
	friend struct FlatVector;
	friend struct ListVector;
	friend struct StringVector;
	friend struct FSSTVector;
	friend struct StructVector;
	friend struct UnionVector;
	friend struct SequenceVector;

	friend class DataChunk;
	friend class VectorCacheBuffer;

public:
	//! Create a vector that references the other vector
	DUCKDB_API Vector(Vector &other);
	//! Create a vector that slices another vector
	DUCKDB_API explicit Vector(Vector &other, const SelectionVector &sel, idx_t count);
	//! Create a vector that slices another vector between a pair of offsets
	DUCKDB_API explicit Vector(Vector &other, idx_t offset, idx_t end);
	//! Create a vector of size one holding the passed on value
	DUCKDB_API explicit Vector(const Value &value);
	//! Create a vector of size tuple_count (non-standard)
	DUCKDB_API explicit Vector(LogicalType type, idx_t capacity = STANDARD_VECTOR_SIZE);
	//! Create an empty standard vector with a type, equivalent to calling Vector(type, true, false)
	DUCKDB_API explicit Vector(const VectorCache &cache);
	//! Create a non-owning vector that references the specified data
	DUCKDB_API Vector(LogicalType type, data_ptr_t dataptr);
	//! Create an owning vector that holds at most STANDARD_VECTOR_SIZE entries.
	/*!
	    Create a new vector
	    If create_data is true, the vector will be an owning empty vector.
	    If zero_data is true, the allocated data will be zero-initialized.
	*/
	DUCKDB_API Vector(LogicalType type, bool create_data, bool zero_data, idx_t capacity = STANDARD_VECTOR_SIZE);
	// implicit copying of Vectors is not allowed
	Vector(const Vector &) = delete;
	// but moving of vectors is allowed
	DUCKDB_API Vector(Vector &&other) noexcept;

public:
	//! Create a vector that references the specified value.
	DUCKDB_API void Reference(const Value &value);
	//! Causes this vector to reference the data held by the other vector.
	//! The type of the "other" vector should match the type of this vector
	DUCKDB_API void Reference(Vector &other);
	//! Reinterpret the data of the other vector as the type of this vector
	//! Note that this takes the data of the other vector as-is and places it in this vector
	//! Without changing the type of this vector
	DUCKDB_API void Reinterpret(Vector &other);

	//! Causes this vector to reference the data held by the other vector, changes the type if required.
	DUCKDB_API void ReferenceAndSetType(Vector &other);

	//! Resets a vector from a vector cache.
	//! This turns the vector back into an empty FlatVector with STANDARD_VECTOR_SIZE entries.
	//! The VectorCache is used so this can be done without requiring any allocations.
	DUCKDB_API void ResetFromCache(const VectorCache &cache);

	//! Creates a reference to a slice of the other vector
	DUCKDB_API void Slice(Vector &other, idx_t offset, idx_t end);
	//! Creates a reference to a slice of the other vector
	DUCKDB_API void Slice(Vector &other, const SelectionVector &sel, idx_t count);
	//! Turns the vector into a dictionary vector with the specified dictionary
	DUCKDB_API void Slice(const SelectionVector &sel, idx_t count);
	//! Slice the vector, keeping the result around in a cache or potentially using the cache instead of slicing
	DUCKDB_API void Slice(const SelectionVector &sel, idx_t count, SelCache &cache);

	//! Creates the data of this vector with the specified type. Any data that
	//! is currently in the vector is destroyed.
	DUCKDB_API void Initialize(bool zero_data = false, idx_t capacity = STANDARD_VECTOR_SIZE);

	//! Converts this Vector to a printable string representation
	DUCKDB_API string ToString(idx_t count) const;
	DUCKDB_API void Print(idx_t count) const;

	DUCKDB_API string ToString() const;
	DUCKDB_API void Print() const;

	//! Flatten the vector, removing any compression and turning it into a FLAT_VECTOR
	DUCKDB_API void Flatten(idx_t count);
	DUCKDB_API void Flatten(const SelectionVector &sel, idx_t count);
	//! Creates a UnifiedVectorFormat of a vector
	//! The UnifiedVectorFormat allows efficient reading of vectors regardless of their vector type
	//! It contains (1) a data pointer, (2) a validity mask, and (3) a selection vector
	//! Access to the individual vector elements can be performed through data_pointer[sel_idx[i]]/validity[sel_idx[i]]
	//! The most common vector types (flat, constant & dictionary) can be converted to the canonical format "for free"
	//! ToUnifiedFormat was originally called Orrify, as a tribute to Orri Erling who came up with it
	DUCKDB_API void ToUnifiedFormat(idx_t count, UnifiedVectorFormat &data);

	//! Turn the vector into a sequence vector
	DUCKDB_API void Sequence(int64_t start, int64_t increment, idx_t count);

	//! Verify that the Vector is in a consistent, not corrupt state. DEBUG
	//! FUNCTION ONLY!
	DUCKDB_API void Verify(idx_t count);
	//! Asserts that the CheckMapValidity returns MapInvalidReason::VALID
	DUCKDB_API static void VerifyMap(Vector &map, const SelectionVector &sel, idx_t count);
	DUCKDB_API static void VerifyUnion(Vector &map, const SelectionVector &sel, idx_t count);
	DUCKDB_API static void Verify(Vector &vector, const SelectionVector &sel, idx_t count);
	DUCKDB_API void UTFVerify(idx_t count);
	DUCKDB_API void UTFVerify(const SelectionVector &sel, idx_t count);

	//! Returns the [index] element of the Vector as a Value.
	DUCKDB_API Value GetValue(idx_t index) const;
	//! Sets the [index] element of the Vector to the specified Value.
	DUCKDB_API void SetValue(idx_t index, const Value &val);

	inline void SetAuxiliary(buffer_ptr<VectorBuffer> new_buffer) {
		auxiliary = std::move(new_buffer);
	};

	//! This functions resizes the vector
	DUCKDB_API void Resize(idx_t cur_size, idx_t new_size);

	//! Serializes a Vector to a stand-alone binary blob
	DUCKDB_API void Serialize(idx_t count, Serializer &serializer);
	//! Deserializes a blob back into a Vector
	DUCKDB_API void Deserialize(idx_t count, Deserializer &source);

	DUCKDB_API void FormatSerialize(FormatSerializer &serializer, idx_t count);
	DUCKDB_API void FormatDeserialize(FormatDeserializer &deserializer, idx_t count);

	// Getters
	inline VectorType GetVectorType() const {
		return vector_type;
	}
	inline const LogicalType &GetType() const {
		return type;
	}
	inline data_ptr_t GetData() {
		return data;
	}

	inline buffer_ptr<VectorBuffer> GetAuxiliary() {
		return auxiliary;
	}

	inline buffer_ptr<VectorBuffer> GetBuffer() {
		return buffer;
	}

	// Setters
	DUCKDB_API void SetVectorType(VectorType vector_type);

private:
	//! Returns the [index] element of the Vector as a Value.
	static Value GetValue(const Vector &v, idx_t index);
	//! Returns the [index] element of the Vector as a Value.
	static Value GetValueInternal(const Vector &v, idx_t index);

protected:
	//! The vector type specifies how the data of the vector is physically stored (i.e. if it is a single repeated
	//! constant, if it is compressed)
	VectorType vector_type;
	//! The type of the elements stored in the vector (e.g. integer, float)
	LogicalType type;
	//! A pointer to the data.
	data_ptr_t data;
	//! The validity mask of the vector
	ValidityMask validity;
	//! The main buffer holding the data of the vector
	buffer_ptr<VectorBuffer> buffer;
	//! The buffer holding auxiliary data of the vector
	//! e.g. a string vector uses this to store strings
	buffer_ptr<VectorBuffer> auxiliary;
};

//! The DictionaryBuffer holds a selection vector
class VectorChildBuffer : public VectorBuffer {
public:
	explicit VectorChildBuffer(Vector vector)
	    : VectorBuffer(VectorBufferType::VECTOR_CHILD_BUFFER), data(std::move(vector)) {
	}

public:
	Vector data;
};

struct ConstantVector {
	static inline const_data_ptr_t GetData(const Vector &vector) {
		D_ASSERT(vector.GetVectorType() == VectorType::CONSTANT_VECTOR ||
		         vector.GetVectorType() == VectorType::FLAT_VECTOR);
		return vector.data;
	}
	static inline data_ptr_t GetData(Vector &vector) {
		D_ASSERT(vector.GetVectorType() == VectorType::CONSTANT_VECTOR ||
		         vector.GetVectorType() == VectorType::FLAT_VECTOR);
		return vector.data;
	}
	template <class T>
	static inline const T *GetData(const Vector &vector) {
		return (const T *)ConstantVector::GetData(vector);
	}
	template <class T>
	static inline T *GetData(Vector &vector) {
		return (T *)ConstantVector::GetData(vector);
	}
	static inline bool IsNull(const Vector &vector) {
		D_ASSERT(vector.GetVectorType() == VectorType::CONSTANT_VECTOR);
		return !vector.validity.RowIsValid(0);
	}
	DUCKDB_API static void SetNull(Vector &vector, bool is_null);
	static inline ValidityMask &Validity(Vector &vector) {
		D_ASSERT(vector.GetVectorType() == VectorType::CONSTANT_VECTOR);
		return vector.validity;
	}
	DUCKDB_API static const SelectionVector *ZeroSelectionVector(idx_t count, SelectionVector &owned_sel);
	DUCKDB_API static const SelectionVector *ZeroSelectionVector();
	//! Turns "vector" into a constant vector by referencing a value within the source vector
	DUCKDB_API static void Reference(Vector &vector, Vector &source, idx_t position, idx_t count);

	static const sel_t ZERO_VECTOR[STANDARD_VECTOR_SIZE];
};

struct DictionaryVector {
	static inline const SelectionVector &SelVector(const Vector &vector) {
		D_ASSERT(vector.GetVectorType() == VectorType::DICTIONARY_VECTOR);
		return ((const DictionaryBuffer &)*vector.buffer).GetSelVector();
	}
	static inline SelectionVector &SelVector(Vector &vector) {
		D_ASSERT(vector.GetVectorType() == VectorType::DICTIONARY_VECTOR);
		return ((DictionaryBuffer &)*vector.buffer).GetSelVector();
	}
	static inline const Vector &Child(const Vector &vector) {
		D_ASSERT(vector.GetVectorType() == VectorType::DICTIONARY_VECTOR);
		return ((const VectorChildBuffer &)*vector.auxiliary).data;
	}
	static inline Vector &Child(Vector &vector) {
		D_ASSERT(vector.GetVectorType() == VectorType::DICTIONARY_VECTOR);
		return ((VectorChildBuffer &)*vector.auxiliary).data;
	}
};

struct FlatVector {
	static inline data_ptr_t GetData(Vector &vector) {
		return ConstantVector::GetData(vector);
	}
	template <class T>
	static inline const T *GetData(const Vector &vector) {
		return ConstantVector::GetData<T>(vector);
	}
	template <class T>
	static inline T *GetData(Vector &vector) {
		return ConstantVector::GetData<T>(vector);
	}
	static inline void SetData(Vector &vector, data_ptr_t data) {
		D_ASSERT(vector.GetVectorType() == VectorType::FLAT_VECTOR);
		vector.data = data;
	}
	template <class T>
	static inline T GetValue(Vector &vector, idx_t idx) {
		D_ASSERT(vector.GetVectorType() == VectorType::FLAT_VECTOR);
		return FlatVector::GetData<T>(vector)[idx];
	}
	static inline const ValidityMask &Validity(const Vector &vector) {
		D_ASSERT(vector.GetVectorType() == VectorType::FLAT_VECTOR);
		return vector.validity;
	}
	static inline ValidityMask &Validity(Vector &vector) {
		D_ASSERT(vector.GetVectorType() == VectorType::FLAT_VECTOR);
		return vector.validity;
	}
	static inline void SetValidity(Vector &vector, ValidityMask &new_validity) {
		D_ASSERT(vector.GetVectorType() == VectorType::FLAT_VECTOR);
		vector.validity.Initialize(new_validity);
	}
	DUCKDB_API static void SetNull(Vector &vector, idx_t idx, bool is_null);
	static inline bool IsNull(const Vector &vector, idx_t idx) {
		D_ASSERT(vector.GetVectorType() == VectorType::FLAT_VECTOR);
		return !vector.validity.RowIsValid(idx);
	}
	DUCKDB_API static const SelectionVector *IncrementalSelectionVector();
};

struct ListVector {
	static inline list_entry_t *GetData(Vector &v) {
		if (v.GetVectorType() == VectorType::DICTIONARY_VECTOR) {
			auto &child = DictionaryVector::Child(v);
			return GetData(child);
		}
		return FlatVector::GetData<list_entry_t>(v);
	}
	//! Gets a reference to the underlying child-vector of a list
	DUCKDB_API static const Vector &GetEntry(const Vector &vector);
	//! Gets a reference to the underlying child-vector of a list
	DUCKDB_API static Vector &GetEntry(Vector &vector);
	//! Gets the total size of the underlying child-vector of a list
	DUCKDB_API static idx_t GetListSize(const Vector &vector);
	//! Sets the total size of the underlying child-vector of a list
	DUCKDB_API static void SetListSize(Vector &vec, idx_t size);
	//! Gets the total capacity of the underlying child-vector of a list
	DUCKDB_API static idx_t GetListCapacity(const Vector &vector);
	//! Sets the total capacity of the underlying child-vector of a list
	DUCKDB_API static void Reserve(Vector &vec, idx_t required_capacity);
	DUCKDB_API static void Append(Vector &target, const Vector &source, idx_t source_size, idx_t source_offset = 0);
	DUCKDB_API static void Append(Vector &target, const Vector &source, const SelectionVector &sel, idx_t source_size,
	                              idx_t source_offset = 0);
	DUCKDB_API static void PushBack(Vector &target, const Value &insert);
	//! Returns the child_vector of list starting at offset until offset + count, and its length
	DUCKDB_API static idx_t GetConsecutiveChildList(Vector &list, Vector &result, idx_t offset, idx_t count);
	//! Returns information to only copy a section of a list child vector
	DUCKDB_API static ConsecutiveChildListInfo GetConsecutiveChildListInfo(Vector &list, idx_t offset, idx_t count);
	//! Slice and flatten a child vector to only contain a consecutive subsection of the child entries
	DUCKDB_API static void GetConsecutiveChildSelVector(Vector &list, SelectionVector &sel, idx_t offset, idx_t count);
	//! Share the entry of the other list vector
	DUCKDB_API static void ReferenceEntry(Vector &vector, Vector &other);
};

struct StringVector {
	//! Add a string to the string heap of the vector (auxiliary data)
	DUCKDB_API static string_t AddString(Vector &vector, const char *data, idx_t len);
	//! Add a string or a blob to the string heap of the vector (auxiliary data)
	//! This function is the same as ::AddString, except the added data does not need to be valid UTF8
	DUCKDB_API static string_t AddStringOrBlob(Vector &vector, const char *data, idx_t len);
	//! Add a string to the string heap of the vector (auxiliary data)
	DUCKDB_API static string_t AddString(Vector &vector, const char *data);
	//! Add a string to the string heap of the vector (auxiliary data)
	DUCKDB_API static string_t AddString(Vector &vector, string_t data);
	//! Add a string to the string heap of the vector (auxiliary data)
	DUCKDB_API static string_t AddString(Vector &vector, const string &data);
	//! Add a string or a blob to the string heap of the vector (auxiliary data)
	//! This function is the same as ::AddString, except the added data does not need to be valid UTF8
	DUCKDB_API static string_t AddStringOrBlob(Vector &vector, string_t data);
	//! Allocates an empty string of the specified size, and returns a writable pointer that can be used to store the
	//! result of an operation
	DUCKDB_API static string_t EmptyString(Vector &vector, idx_t len);
	//! Adds a reference to a handle that stores strings of this vector
	DUCKDB_API static void AddHandle(Vector &vector, BufferHandle handle);
	//! Adds a reference to an unspecified vector buffer that stores strings of this vector
	DUCKDB_API static void AddBuffer(Vector &vector, buffer_ptr<VectorBuffer> buffer);
	//! Add a reference from this vector to the string heap of the provided vector
	DUCKDB_API static void AddHeapReference(Vector &vector, Vector &other);
};

struct FSSTVector {
	static inline const ValidityMask &Validity(const Vector &vector) {
		D_ASSERT(vector.GetVectorType() == VectorType::FSST_VECTOR);
		return vector.validity;
	}
	static inline ValidityMask &Validity(Vector &vector) {
		D_ASSERT(vector.GetVectorType() == VectorType::FSST_VECTOR);
		return vector.validity;
	}
	static inline void SetValidity(Vector &vector, ValidityMask &new_validity) {
		D_ASSERT(vector.GetVectorType() == VectorType::FSST_VECTOR);
		vector.validity.Initialize(new_validity);
	}
	static inline const_data_ptr_t GetCompressedData(const Vector &vector) {
		D_ASSERT(vector.GetVectorType() == VectorType::FSST_VECTOR);
		return vector.data;
	}
	static inline data_ptr_t GetCompressedData(Vector &vector) {
		D_ASSERT(vector.GetVectorType() == VectorType::FSST_VECTOR);
		return vector.data;
	}
	template <class T>
	static inline const T *GetCompressedData(const Vector &vector) {
		return (const T *)FSSTVector::GetCompressedData(vector);
	}
	template <class T>
	static inline T *GetCompressedData(Vector &vector) {
		return (T *)FSSTVector::GetCompressedData(vector);
	}
	//! Decompresses an FSST_VECTOR into a FLAT_VECTOR. Note: validity is not copied.
	static void DecompressVector(const Vector &src, Vector &dst, idx_t src_offset, idx_t dst_offset, idx_t copy_count,
	                             const SelectionVector *sel);

	DUCKDB_API static string_t AddCompressedString(Vector &vector, string_t data);
	DUCKDB_API static string_t AddCompressedString(Vector &vector, const char *data, idx_t len);
	DUCKDB_API static void RegisterDecoder(Vector &vector, buffer_ptr<void> &duckdb_fsst_decoder);
	DUCKDB_API static void *GetDecoder(const Vector &vector);
	//! Setting the string count is required to be able to correctly flatten the vector
	DUCKDB_API static void SetCount(Vector &vector, idx_t count);
	DUCKDB_API static idx_t GetCount(Vector &vector);
};

enum class MapInvalidReason : uint8_t { VALID, NULL_KEY_LIST, NULL_KEY, DUPLICATE_KEY };

struct MapVector {
	DUCKDB_API static const Vector &GetKeys(const Vector &vector);
	DUCKDB_API static const Vector &GetValues(const Vector &vector);
	DUCKDB_API static Vector &GetKeys(Vector &vector);
	DUCKDB_API static Vector &GetValues(Vector &vector);
	DUCKDB_API static MapInvalidReason
	CheckMapValidity(Vector &map, idx_t count, const SelectionVector &sel = *FlatVector::IncrementalSelectionVector());
	DUCKDB_API static void MapConversionVerify(Vector &vector, idx_t count);
};

struct StructVector {
	DUCKDB_API static const vector<unique_ptr<Vector>> &GetEntries(const Vector &vector);
	DUCKDB_API static vector<unique_ptr<Vector>> &GetEntries(Vector &vector);
};

enum class UnionInvalidReason : uint8_t { VALID, TAG_OUT_OF_RANGE, NO_MEMBERS, VALIDITY_OVERLAP };

struct UnionVector {
	// Unions are stored as structs, but the first child is always the "tag"
	// vector, specifying the currently selected member for that row.
	// The remaining children are the members of the union.
	// INVARIANTS:
	//	1.	Only one member vector (the one "selected" by the tag) can be
	//		non-NULL in each row.
	//
	//	2.	The validity of the tag vector always matches the validity of the
	//		union vector itself.
	//
	//	3.	For each tag in the tag vector, 0 <= tag < |members|

	//! Get the tag vector of a union vector
	DUCKDB_API static const Vector &GetTags(const Vector &v);
	DUCKDB_API static Vector &GetTags(Vector &v);

	//! Get the tag at the specific index of the union vector
	DUCKDB_API static union_tag_t GetTag(const Vector &vector, idx_t index);

	//! Get the member vector of a union vector by index
	DUCKDB_API static const Vector &GetMember(const Vector &vector, idx_t member_index);
	DUCKDB_API static Vector &GetMember(Vector &vector, idx_t member_index);

	//! Set every entry in the UnionVector to a specific member.
	//! This is useful to set the entire vector to a single member, e.g. when "creating"
	//! a union to return in a function, when you only have one alternative to return.
	//! if 'keep_tags_for_null' is false, the tags will be set to NULL where the member is NULL.
	//! (the validity of the tag vector will match the selected member vector)
	//! otherwise, they are all set to the 'tag'.
	//! This will also handle invalidation of the non-selected members
	DUCKDB_API static void SetToMember(Vector &vector, union_tag_t tag, Vector &member_vector, idx_t count,
	                                   bool keep_tags_for_null);

	DUCKDB_API static UnionInvalidReason
	CheckUnionValidity(Vector &vector, idx_t count,
	                   const SelectionVector &sel = *FlatVector::IncrementalSelectionVector());
};

struct SequenceVector {
	static void GetSequence(const Vector &vector, int64_t &start, int64_t &increment, int64_t &sequence_count) {
		D_ASSERT(vector.GetVectorType() == VectorType::SEQUENCE_VECTOR);
		auto data = (int64_t *)vector.buffer->GetData();
		start = data[0];
		increment = data[1];
		sequence_count = data[2];
	}
	static void GetSequence(const Vector &vector, int64_t &start, int64_t &increment) {
		int64_t sequence_count;
		GetSequence(vector, start, increment, sequence_count);
	}
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/vector_operations/vector_operations.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/types/data_chunk.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/arrow/arrow_wrapper.hpp
//
//
//===----------------------------------------------------------------------===//


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/arrow/arrow.hpp
//
//
//===----------------------------------------------------------------------===//

#ifndef ARROW_FLAG_DICTIONARY_ORDERED

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

#ifndef ARROW_C_DATA_INTERFACE
#define ARROW_C_DATA_INTERFACE

#define ARROW_FLAG_DICTIONARY_ORDERED 1
#define ARROW_FLAG_NULLABLE           2
#define ARROW_FLAG_MAP_KEYS_SORTED    4

struct ArrowSchema {
	// Array type description
	const char *format;
	const char *name;
	const char *metadata;
	int64_t flags;
	int64_t n_children;
	struct ArrowSchema **children;
	struct ArrowSchema *dictionary;

	// Release callback
	void (*release)(struct ArrowSchema *);
	// Opaque producer-specific data
	void *private_data;
};

struct ArrowArray {
	// Array data description
	int64_t length;
	int64_t null_count;
	int64_t offset;
	int64_t n_buffers;
	int64_t n_children;
	const void **buffers;
	struct ArrowArray **children;
	struct ArrowArray *dictionary;

	// Release callback
	void (*release)(struct ArrowArray *);
	// Opaque producer-specific data
	void *private_data;
};
#endif

#ifndef ARROW_C_STREAM_INTERFACE
#define ARROW_C_STREAM_INTERFACE
// EXPERIMENTAL
struct ArrowArrayStream {
	// Callback to get the stream type
	// (will be the same for all arrays in the stream).
	// Return value: 0 if successful, an `errno`-compatible error code otherwise.
	int (*get_schema)(struct ArrowArrayStream *, struct ArrowSchema *out);
	// Callback to get the next array
	// (if no error and the array is released, the stream has ended)
	// Return value: 0 if successful, an `errno`-compatible error code otherwise.
	int (*get_next)(struct ArrowArrayStream *, struct ArrowArray *out);

	// Callback to get optional detailed error information.
	// This must only be called if the last stream operation failed
	// with a non-0 return code.  The returned pointer is only valid until
	// the next operation on this stream (including release).
	// If unavailable, NULL is returned.
	const char *(*get_last_error)(struct ArrowArrayStream *);

	// Release callback: release the stream's own resources.
	// Note that arrays returned by `get_next` must be individually released.
	void (*release)(struct ArrowArrayStream *);
	// Opaque producer-specific data
	void *private_data;
};
#endif

#ifdef __cplusplus
}
#endif

#endif


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/preserved_error.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {

class PreservedError {
public:
	//! Not initialized, default constructor
	DUCKDB_API PreservedError();
	//! From std::exception
	PreservedError(const std::exception &ex)
	    : initialized(true), type(ExceptionType::INVALID), raw_message(SanitizeErrorMessage(ex.what())),
	      exception_instance(nullptr) {
	}
	//! From a raw string
	DUCKDB_API explicit PreservedError(const string &raw_message);
	//! From an Exception
	DUCKDB_API PreservedError(const Exception &exception);

public:
	//! Throw the error
	[[noreturn]] DUCKDB_API void Throw(const string &prepended_message = "") const;
	//! Get the internal exception type of the error
	DUCKDB_API const ExceptionType &Type() const;
	//! Allows adding addition information to the message
	DUCKDB_API PreservedError &AddToMessage(const string &prepended_message);
	//! Used in clients like C-API, creates the final message and returns a reference to it
	DUCKDB_API const string &Message();
	//! Let's us do things like 'if (error)'
	DUCKDB_API operator bool() const;
	DUCKDB_API bool operator==(const PreservedError &other) const;
	const shared_ptr<Exception> &GetError() {
		return exception_instance;
	}

private:
	//! Whether this PreservedError contains an exception or not
	bool initialized;
	//! The ExceptionType of the preserved exception
	ExceptionType type;
	//! The message the exception was constructed with (does not contain the Exception Type)
	string raw_message;
	//! The final message (stored in the preserved error for compatibility reasons with C-API)
	string final_message;
	std::shared_ptr<Exception> exception_instance;

private:
	DUCKDB_API static string SanitizeErrorMessage(string error);
};

} // namespace duckdb


//! Here we have the internal duckdb classes that interact with Arrow's Internal Header (i.e., duckdb/commons/arrow.hpp)
namespace duckdb {
class QueryResult;
class DataChunk;

class ArrowSchemaWrapper {
public:
	ArrowSchema arrow_schema;

	ArrowSchemaWrapper() {
		arrow_schema.release = nullptr;
	}

	~ArrowSchemaWrapper();
};
class ArrowArrayWrapper {
public:
	ArrowArray arrow_array;
	ArrowArrayWrapper() {
		arrow_array.length = 0;
		arrow_array.release = nullptr;
	}
	~ArrowArrayWrapper();
};

class ArrowArrayStreamWrapper {
public:
	ArrowArrayStream arrow_array_stream;
	int64_t number_of_rows;

public:
	void GetSchema(ArrowSchemaWrapper &schema);

	shared_ptr<ArrowArrayWrapper> GetNextChunk();

	const char *GetError();

	~ArrowArrayStreamWrapper();
	ArrowArrayStreamWrapper() {
		arrow_array_stream.release = nullptr;
	}
};

class ArrowUtil {
public:
	static bool TryFetchChunk(QueryResult *result, idx_t chunk_size, ArrowArray *out, idx_t &result_count,
	                          PreservedError &error);
	static idx_t FetchChunk(QueryResult *result, idx_t chunk_size, ArrowArray *out);

private:
	static bool TryFetchNext(QueryResult &result, unique_ptr<DataChunk> &out, PreservedError &error);
};
} // namespace duckdb





struct ArrowArray;

namespace duckdb {
class Allocator;
class ClientContext;
class ExecutionContext;
class VectorCache;

//!  A Data Chunk represents a set of vectors.
/*!
    The data chunk class is the intermediate representation used by the
   execution engine of DuckDB. It effectively represents a subset of a relation.
   It holds a set of vectors that all have the same length.

    DataChunk is initialized using the DataChunk::Initialize function by
   providing it with a vector of TypeIds for the Vector members. By default,
   this function will also allocate a chunk of memory in the DataChunk for the
   vectors and all the vectors will be referencing vectors to the data owned by
   the chunk. The reason for this behavior is that the underlying vectors can
   become referencing vectors to other chunks as well (i.e. in the case an
   operator does not alter the data, such as a Filter operator which only adds a
   selection vector).

    In addition to holding the data of the vectors, the DataChunk also owns the
   selection vector that underlying vectors can point to.
*/
class DataChunk {
public:
	//! Creates an empty DataChunk
	DUCKDB_API DataChunk();
	DUCKDB_API ~DataChunk();

	//! The vectors owned by the DataChunk.
	vector<Vector> data;

public:
	inline idx_t size() const { // NOLINT
		return count;
	}
	inline idx_t ColumnCount() const {
		return data.size();
	}
	inline void SetCardinality(idx_t count_p) {
		D_ASSERT(count_p <= capacity);
		this->count = count_p;
	}
	inline void SetCardinality(const DataChunk &other) {
		SetCardinality(other.size());
	}
	inline void SetCapacity(idx_t capacity_p) {
		this->capacity = capacity_p;
	}
	inline void SetCapacity(const DataChunk &other) {
		SetCapacity(other.capacity);
	}

	DUCKDB_API Value GetValue(idx_t col_idx, idx_t index) const;
	DUCKDB_API void SetValue(idx_t col_idx, idx_t index, const Value &val);

	//! Returns true if all vectors in the DataChunk are constant
	DUCKDB_API bool AllConstant() const;

	//! Set the DataChunk to reference another data chunk
	DUCKDB_API void Reference(DataChunk &chunk);
	//! Set the DataChunk to own the data of data chunk, destroying the other chunk in the process
	DUCKDB_API void Move(DataChunk &chunk);

	//! Initializes the DataChunk with the specified types to an empty DataChunk
	//! This will create one vector of the specified type for each LogicalType in the
	//! types list. The vector will be referencing vector to the data owned by
	//! the DataChunk.
	DUCKDB_API void Initialize(Allocator &allocator, const vector<LogicalType> &types,
	                           idx_t capacity = STANDARD_VECTOR_SIZE);
	DUCKDB_API void Initialize(ClientContext &context, const vector<LogicalType> &types,
	                           idx_t capacity = STANDARD_VECTOR_SIZE);
	//! Initializes an empty DataChunk with the given types. The vectors will *not* have any data allocated for them.
	DUCKDB_API void InitializeEmpty(const vector<LogicalType> &types);

	DUCKDB_API void InitializeEmpty(vector<LogicalType>::const_iterator begin, vector<LogicalType>::const_iterator end);
	DUCKDB_API void Initialize(Allocator &allocator, vector<LogicalType>::const_iterator begin,
	                           vector<LogicalType>::const_iterator end, idx_t capacity = STANDARD_VECTOR_SIZE);
	DUCKDB_API void Initialize(ClientContext &context, vector<LogicalType>::const_iterator begin,
	                           vector<LogicalType>::const_iterator end, idx_t capacity = STANDARD_VECTOR_SIZE);

	//! Append the other DataChunk to this one. The column count and types of
	//! the two DataChunks have to match exactly. Throws an exception if there
	//! is not enough space in the chunk and resize is not allowed.
	DUCKDB_API void Append(const DataChunk &other, bool resize = false, SelectionVector *sel = nullptr,
	                       idx_t count = 0);

	//! Destroy all data and columns owned by this DataChunk
	DUCKDB_API void Destroy();

	//! Copies the data from this vector to another vector.
	DUCKDB_API void Copy(DataChunk &other, idx_t offset = 0) const;
	DUCKDB_API void Copy(DataChunk &other, const SelectionVector &sel, const idx_t source_count,
	                     const idx_t offset = 0) const;

	//! Splits the DataChunk in two
	DUCKDB_API void Split(DataChunk &other, idx_t split_idx);

	//! Fuses a DataChunk onto the right of this one, and destroys the other. Inverse of Split.
	DUCKDB_API void Fuse(DataChunk &other);

	//! Makes this DataChunk reference the specified columns in the other DataChunk
	DUCKDB_API void ReferenceColumns(DataChunk &other, const vector<column_t> &column_ids);

	//! Turn all the vectors from the chunk into flat vectors
	DUCKDB_API void Flatten();

	// FIXME: this is DUCKDB_API, might need conversion back to regular unique ptr?
	DUCKDB_API unsafe_unique_array<UnifiedVectorFormat> ToUnifiedFormat();

	DUCKDB_API void Slice(const SelectionVector &sel_vector, idx_t count);

	//! Slice all Vectors from other.data[i] to data[i + 'col_offset']
	//! Turning all Vectors into Dictionary Vectors, using 'sel'
	DUCKDB_API void Slice(DataChunk &other, const SelectionVector &sel, idx_t count, idx_t col_offset = 0);

	//! Resets the DataChunk to its state right after the DataChunk::Initialize
	//! function was called. This sets the count to 0, and resets each member
	//! Vector to point back to the data owned by this DataChunk.
	DUCKDB_API void Reset();

	//! Serializes a DataChunk to a stand-alone binary blob
	DUCKDB_API void Serialize(Serializer &serializer);
	//! Deserializes a blob back into a DataChunk
	DUCKDB_API void Deserialize(Deserializer &source);

	//! Hashes the DataChunk to the target vector
	DUCKDB_API void Hash(Vector &result);
	//! Hashes specific vectors of the DataChunk to the target vector
	DUCKDB_API void Hash(vector<idx_t> &column_ids, Vector &result);

	//! Returns a list of types of the vectors of this data chunk
	DUCKDB_API vector<LogicalType> GetTypes();

	//! Converts this DataChunk to a printable string representation
	DUCKDB_API string ToString() const;
	DUCKDB_API void Print();

	DataChunk(const DataChunk &) = delete;

	//! Verify that the DataChunk is in a consistent, not corrupt state. DEBUG
	//! FUNCTION ONLY!
	DUCKDB_API void Verify();

private:
	//! The amount of tuples stored in the data chunk
	idx_t count;
	//! The amount of tuples that can be stored in the data chunk
	idx_t capacity;
	//! Vector caches, used to store data when ::Initialize is called
	vector<VectorCache> vector_caches;
};
} // namespace duckdb



#include <functional>

namespace duckdb {
class CastFunctionSet;
struct GetCastFunctionInput;

// VectorOperations contains a set of operations that operate on sets of
// vectors. In general, the operators must all have the same type, otherwise an
// exception is thrown. Note that the functions underneath use restrict
// pointers, hence the data that the vectors point to (and hence the vector
// themselves) should not be equal! For example, if you call the function Add(A,
// B, A) then ASSERT_RESTRICT will be triggered. Instead call AddInPlace(A, B)
// or Add(A, B, C)
struct VectorOperations {
	//===--------------------------------------------------------------------===//
	// In-Place Operators
	//===--------------------------------------------------------------------===//
	//! left += delta
	static void AddInPlace(Vector &left, int64_t delta, idx_t count);

	//===--------------------------------------------------------------------===//
	// NULL Operators
	//===--------------------------------------------------------------------===//
	//! result = IS NOT NULL(input)
	static void IsNotNull(Vector &arg, Vector &result, idx_t count);
	//! result = IS NULL (input)
	static void IsNull(Vector &input, Vector &result, idx_t count);
	// Returns whether or not arg vector has a NULL value
	static bool HasNull(Vector &input, idx_t count);
	static bool HasNotNull(Vector &input, idx_t count);
	//! Count the number of not-NULL values.
	static idx_t CountNotNull(Vector &input, const idx_t count);

	//===--------------------------------------------------------------------===//
	// Boolean Operations
	//===--------------------------------------------------------------------===//
	// result = left && right
	static void And(Vector &left, Vector &right, Vector &result, idx_t count);
	// result = left || right
	static void Or(Vector &left, Vector &right, Vector &result, idx_t count);
	// result = NOT(left)
	static void Not(Vector &left, Vector &result, idx_t count);

	//===--------------------------------------------------------------------===//
	// Comparison Operations
	//===--------------------------------------------------------------------===//
	// result = left == right
	static void Equals(Vector &left, Vector &right, Vector &result, idx_t count);
	// result = left != right
	static void NotEquals(Vector &left, Vector &right, Vector &result, idx_t count);
	// result = left > right
	static void GreaterThan(Vector &left, Vector &right, Vector &result, idx_t count);
	// result = left >= right
	static void GreaterThanEquals(Vector &left, Vector &right, Vector &result, idx_t count);
	// result = left < right
	static void LessThan(Vector &left, Vector &right, Vector &result, idx_t count);
	// result = left <= right
	static void LessThanEquals(Vector &left, Vector &right, Vector &result, idx_t count);

	// result = A != B with nulls being equal
	static void DistinctFrom(Vector &left, Vector &right, Vector &result, idx_t count);
	// result := A == B with nulls being equal
	static void NotDistinctFrom(Vector &left, Vector &right, Vector &result, idx_t count);
	// result := A > B with nulls being maximal
	static void DistinctGreaterThan(Vector &left, Vector &right, Vector &result, idx_t count);
	// result := A >= B with nulls being maximal
	static void DistinctGreaterThanEquals(Vector &left, Vector &right, Vector &result, idx_t count);
	// result := A < B with nulls being maximal
	static void DistinctLessThan(Vector &left, Vector &right, Vector &result, idx_t count);
	// result := A <= B with nulls being maximal
	static void DistinctLessThanEquals(Vector &left, Vector &right, Vector &result, idx_t count);

	//===--------------------------------------------------------------------===//
	// Select Comparisons
	//===--------------------------------------------------------------------===//
	static idx_t Equals(Vector &left, Vector &right, const SelectionVector *sel, idx_t count, SelectionVector *true_sel,
	                    SelectionVector *false_sel);
	static idx_t NotEquals(Vector &left, Vector &right, const SelectionVector *sel, idx_t count,
	                       SelectionVector *true_sel, SelectionVector *false_sel);
	static idx_t GreaterThan(Vector &left, Vector &right, const SelectionVector *sel, idx_t count,
	                         SelectionVector *true_sel, SelectionVector *false_sel);
	static idx_t GreaterThanEquals(Vector &left, Vector &right, const SelectionVector *sel, idx_t count,
	                               SelectionVector *true_sel, SelectionVector *false_sel);
	static idx_t LessThan(Vector &left, Vector &right, const SelectionVector *sel, idx_t count,
	                      SelectionVector *true_sel, SelectionVector *false_sel);
	static idx_t LessThanEquals(Vector &left, Vector &right, const SelectionVector *sel, idx_t count,
	                            SelectionVector *true_sel, SelectionVector *false_sel);

	// true := A != B with nulls being equal
	static idx_t DistinctFrom(Vector &left, Vector &right, const SelectionVector *sel, idx_t count,
	                          SelectionVector *true_sel, SelectionVector *false_sel);
	// true := A == B with nulls being equal
	static idx_t NotDistinctFrom(Vector &left, Vector &right, const SelectionVector *sel, idx_t count,
	                             SelectionVector *true_sel, SelectionVector *false_sel);
	// true := A > B with nulls being maximal
	static idx_t DistinctGreaterThan(Vector &left, Vector &right, const SelectionVector *sel, idx_t count,
	                                 SelectionVector *true_sel, SelectionVector *false_sel);
	// true := A >= B with nulls being maximal
	static idx_t DistinctGreaterThanEquals(Vector &left, Vector &right, const SelectionVector *sel, idx_t count,
	                                       SelectionVector *true_sel, SelectionVector *false_sel);
	// true := A < B with nulls being maximal
	static idx_t DistinctLessThan(Vector &left, Vector &right, const SelectionVector *sel, idx_t count,
	                              SelectionVector *true_sel, SelectionVector *false_sel);
	// true := A <= B with nulls being maximal
	static idx_t DistinctLessThanEquals(Vector &left, Vector &right, const SelectionVector *sel, idx_t count,
	                                    SelectionVector *true_sel, SelectionVector *false_sel);

	// true := A > B with nulls being minimal
	static idx_t DistinctGreaterThanNullsFirst(Vector &left, Vector &right, const SelectionVector *sel, idx_t count,
	                                           SelectionVector *true_sel, SelectionVector *false_sel);
	// true := A < B with nulls being minimal
	static idx_t DistinctLessThanNullsFirst(Vector &left, Vector &right, const SelectionVector *sel, idx_t count,
	                                        SelectionVector *true_sel, SelectionVector *false_sel);

	//===--------------------------------------------------------------------===//
	// Nested Comparisons
	//===--------------------------------------------------------------------===//
	// true := A != B with nulls being equal
	static idx_t NestedNotEquals(Vector &left, Vector &right, const SelectionVector &sel, idx_t count,
	                             SelectionVector *true_sel, SelectionVector *false_sel);
	// true := A == B with nulls being equal
	static idx_t NestedEquals(Vector &left, Vector &right, const SelectionVector &sel, idx_t count,
	                          SelectionVector *true_sel, SelectionVector *false_sel);

	//===--------------------------------------------------------------------===//
	// Hash functions
	//===--------------------------------------------------------------------===//
	// hashes = HASH(input)
	static void Hash(Vector &input, Vector &hashes, idx_t count);
	static void Hash(Vector &input, Vector &hashes, const SelectionVector &rsel, idx_t count);
	// hashes ^= HASH(input)
	static void CombineHash(Vector &hashes, Vector &input, idx_t count);
	static void CombineHash(Vector &hashes, Vector &input, const SelectionVector &rsel, idx_t count);

	//===--------------------------------------------------------------------===//
	// Generate functions
	//===--------------------------------------------------------------------===//
	static void GenerateSequence(Vector &result, idx_t count, int64_t start = 0, int64_t increment = 1);
	static void GenerateSequence(Vector &result, idx_t count, const SelectionVector &sel, int64_t start = 0,
	                             int64_t increment = 1);
	//===--------------------------------------------------------------------===//
	// Helpers
	//===--------------------------------------------------------------------===//
	//! Cast the data from the source type to the target type. Any elements that could not be converted are turned into
	//! NULLs. If any elements cannot be converted, returns false and fills in the error_message. If no error message is
	//! provided, an exception is thrown instead.
	DUCKDB_API static bool TryCast(CastFunctionSet &set, GetCastFunctionInput &input, Vector &source, Vector &result,
	                               idx_t count, string *error_message, bool strict = false);
	DUCKDB_API static bool DefaultTryCast(Vector &source, Vector &result, idx_t count, string *error_message,
	                                      bool strict = false);
	DUCKDB_API static bool TryCast(ClientContext &context, Vector &source, Vector &result, idx_t count,
	                               string *error_message, bool strict = false);
	//! Cast the data from the source type to the target type. Throws an exception if the cast fails.
	DUCKDB_API static void Cast(ClientContext &context, Vector &source, Vector &result, idx_t count,
	                            bool strict = false);
	DUCKDB_API static void DefaultCast(Vector &source, Vector &result, idx_t count, bool strict = false);

	// Copy the data of <source> to the target vector
	static void Copy(const Vector &source, Vector &target, idx_t source_count, idx_t source_offset,
	                 idx_t target_offset);
	static void Copy(const Vector &source, Vector &target, const SelectionVector &sel, idx_t source_count,
	                 idx_t source_offset, idx_t target_offset);

	// Copy the data of <source> to the target location, setting null values to
	// NullValue<T>. Used to store data without separate NULL mask.
	static void WriteToStorage(Vector &source, idx_t count, data_ptr_t target);
	// Reads the data of <source> to the target vector, setting the nullmask
	// for any NullValue<T> of source. Used to go back from storage to a proper vector
	static void ReadFromStorage(data_ptr_t source, idx_t count, Vector &result);
};
} // namespace duckdb


#include <functional>

namespace duckdb {

struct DefaultNullCheckOperator {
	template <class LEFT_TYPE, class RIGHT_TYPE>
	static inline bool Operation(LEFT_TYPE left, RIGHT_TYPE right) {
		return false;
	}
};

struct BinaryStandardOperatorWrapper {
	template <class FUNC, class OP, class LEFT_TYPE, class RIGHT_TYPE, class RESULT_TYPE>
	static inline RESULT_TYPE Operation(FUNC fun, LEFT_TYPE left, RIGHT_TYPE right, ValidityMask &mask, idx_t idx) {
		return OP::template Operation<LEFT_TYPE, RIGHT_TYPE, RESULT_TYPE>(left, right);
	}

	static bool AddsNulls() {
		return false;
	}
};

struct BinarySingleArgumentOperatorWrapper {
	template <class FUNC, class OP, class LEFT_TYPE, class RIGHT_TYPE, class RESULT_TYPE>
	static inline RESULT_TYPE Operation(FUNC fun, LEFT_TYPE left, RIGHT_TYPE right, ValidityMask &mask, idx_t idx) {
		return OP::template Operation<LEFT_TYPE>(left, right);
	}

	static bool AddsNulls() {
		return false;
	}
};

struct BinaryLambdaWrapper {
	template <class FUNC, class OP, class LEFT_TYPE, class RIGHT_TYPE, class RESULT_TYPE>
	static inline RESULT_TYPE Operation(FUNC fun, LEFT_TYPE left, RIGHT_TYPE right, ValidityMask &mask, idx_t idx) {
		return fun(left, right);
	}

	static bool AddsNulls() {
		return false;
	}
};

struct BinaryLambdaWrapperWithNulls {
	template <class FUNC, class OP, class LEFT_TYPE, class RIGHT_TYPE, class RESULT_TYPE>
	static inline RESULT_TYPE Operation(FUNC fun, LEFT_TYPE left, RIGHT_TYPE right, ValidityMask &mask, idx_t idx) {
		return fun(left, right, mask, idx);
	}

	static bool AddsNulls() {
		return true;
	}
};

struct BinaryExecutor {
	template <class LEFT_TYPE, class RIGHT_TYPE, class RESULT_TYPE, class OPWRAPPER, class OP, class FUNC,
	          bool LEFT_CONSTANT, bool RIGHT_CONSTANT>
	static void ExecuteFlatLoop(const LEFT_TYPE *__restrict ldata, const RIGHT_TYPE *__restrict rdata,
	                            RESULT_TYPE *__restrict result_data, idx_t count, ValidityMask &mask, FUNC fun) {
		if (!LEFT_CONSTANT) {
			ASSERT_RESTRICT(ldata, ldata + count, result_data, result_data + count);
		}
		if (!RIGHT_CONSTANT) {
			ASSERT_RESTRICT(rdata, rdata + count, result_data, result_data + count);
		}

		if (!mask.AllValid()) {
			idx_t base_idx = 0;
			auto entry_count = ValidityMask::EntryCount(count);
			for (idx_t entry_idx = 0; entry_idx < entry_count; entry_idx++) {
				auto validity_entry = mask.GetValidityEntry(entry_idx);
				idx_t next = MinValue<idx_t>(base_idx + ValidityMask::BITS_PER_VALUE, count);
				if (ValidityMask::AllValid(validity_entry)) {
					// all valid: perform operation
					for (; base_idx < next; base_idx++) {
						auto lentry = ldata[LEFT_CONSTANT ? 0 : base_idx];
						auto rentry = rdata[RIGHT_CONSTANT ? 0 : base_idx];
						result_data[base_idx] =
						    OPWRAPPER::template Operation<FUNC, OP, LEFT_TYPE, RIGHT_TYPE, RESULT_TYPE>(
						        fun, lentry, rentry, mask, base_idx);
					}
				} else if (ValidityMask::NoneValid(validity_entry)) {
					// nothing valid: skip all
					base_idx = next;
					continue;
				} else {
					// partially valid: need to check individual elements for validity
					idx_t start = base_idx;
					for (; base_idx < next; base_idx++) {
						if (ValidityMask::RowIsValid(validity_entry, base_idx - start)) {
							auto lentry = ldata[LEFT_CONSTANT ? 0 : base_idx];
							auto rentry = rdata[RIGHT_CONSTANT ? 0 : base_idx];
							result_data[base_idx] =
							    OPWRAPPER::template Operation<FUNC, OP, LEFT_TYPE, RIGHT_TYPE, RESULT_TYPE>(
							        fun, lentry, rentry, mask, base_idx);
						}
					}
				}
			}
		} else {
			for (idx_t i = 0; i < count; i++) {
				auto lentry = ldata[LEFT_CONSTANT ? 0 : i];
				auto rentry = rdata[RIGHT_CONSTANT ? 0 : i];
				result_data[i] = OPWRAPPER::template Operation<FUNC, OP, LEFT_TYPE, RIGHT_TYPE, RESULT_TYPE>(
				    fun, lentry, rentry, mask, i);
			}
		}
	}

	template <class LEFT_TYPE, class RIGHT_TYPE, class RESULT_TYPE, class OPWRAPPER, class OP, class FUNC>
	static void ExecuteConstant(Vector &left, Vector &right, Vector &result, FUNC fun) {
		result.SetVectorType(VectorType::CONSTANT_VECTOR);

		auto ldata = ConstantVector::GetData<LEFT_TYPE>(left);
		auto rdata = ConstantVector::GetData<RIGHT_TYPE>(right);
		auto result_data = ConstantVector::GetData<RESULT_TYPE>(result);

		if (ConstantVector::IsNull(left) || ConstantVector::IsNull(right)) {
			ConstantVector::SetNull(result, true);
			return;
		}
		*result_data = OPWRAPPER::template Operation<FUNC, OP, LEFT_TYPE, RIGHT_TYPE, RESULT_TYPE>(
		    fun, *ldata, *rdata, ConstantVector::Validity(result), 0);
	}

	template <class LEFT_TYPE, class RIGHT_TYPE, class RESULT_TYPE, class OPWRAPPER, class OP, class FUNC,
	          bool LEFT_CONSTANT, bool RIGHT_CONSTANT>
	static void ExecuteFlat(Vector &left, Vector &right, Vector &result, idx_t count, FUNC fun) {
		auto ldata = FlatVector::GetData<LEFT_TYPE>(left);
		auto rdata = FlatVector::GetData<RIGHT_TYPE>(right);

		if ((LEFT_CONSTANT && ConstantVector::IsNull(left)) || (RIGHT_CONSTANT && ConstantVector::IsNull(right))) {
			// either left or right is constant NULL: result is constant NULL
			result.SetVectorType(VectorType::CONSTANT_VECTOR);
			ConstantVector::SetNull(result, true);
			return;
		}

		result.SetVectorType(VectorType::FLAT_VECTOR);
		auto result_data = FlatVector::GetData<RESULT_TYPE>(result);
		auto &result_validity = FlatVector::Validity(result);
		if (LEFT_CONSTANT) {
			if (OPWRAPPER::AddsNulls()) {
				result_validity.Copy(FlatVector::Validity(right), count);
			} else {
				FlatVector::SetValidity(result, FlatVector::Validity(right));
			}
		} else if (RIGHT_CONSTANT) {
			if (OPWRAPPER::AddsNulls()) {
				result_validity.Copy(FlatVector::Validity(left), count);
			} else {
				FlatVector::SetValidity(result, FlatVector::Validity(left));
			}
		} else {
			if (OPWRAPPER::AddsNulls()) {
				result_validity.Copy(FlatVector::Validity(left), count);
				if (result_validity.AllValid()) {
					result_validity.Copy(FlatVector::Validity(right), count);
				} else {
					result_validity.Combine(FlatVector::Validity(right), count);
				}
			} else {
				FlatVector::SetValidity(result, FlatVector::Validity(left));
				result_validity.Combine(FlatVector::Validity(right), count);
			}
		}
		ExecuteFlatLoop<LEFT_TYPE, RIGHT_TYPE, RESULT_TYPE, OPWRAPPER, OP, FUNC, LEFT_CONSTANT, RIGHT_CONSTANT>(
		    ldata, rdata, result_data, count, result_validity, fun);
	}

	template <class LEFT_TYPE, class RIGHT_TYPE, class RESULT_TYPE, class OPWRAPPER, class OP, class FUNC>
	static void ExecuteGenericLoop(const LEFT_TYPE *__restrict ldata, const RIGHT_TYPE *__restrict rdata,
	                               RESULT_TYPE *__restrict result_data, const SelectionVector *__restrict lsel,
	                               const SelectionVector *__restrict rsel, idx_t count, ValidityMask &lvalidity,
	                               ValidityMask &rvalidity, ValidityMask &result_validity, FUNC fun) {
		if (!lvalidity.AllValid() || !rvalidity.AllValid()) {
			for (idx_t i = 0; i < count; i++) {
				auto lindex = lsel->get_index(i);
				auto rindex = rsel->get_index(i);
				if (lvalidity.RowIsValid(lindex) && rvalidity.RowIsValid(rindex)) {
					auto lentry = ldata[lindex];
					auto rentry = rdata[rindex];
					result_data[i] = OPWRAPPER::template Operation<FUNC, OP, LEFT_TYPE, RIGHT_TYPE, RESULT_TYPE>(
					    fun, lentry, rentry, result_validity, i);
				} else {
					result_validity.SetInvalid(i);
				}
			}
		} else {
			for (idx_t i = 0; i < count; i++) {
				auto lentry = ldata[lsel->get_index(i)];
				auto rentry = rdata[rsel->get_index(i)];
				result_data[i] = OPWRAPPER::template Operation<FUNC, OP, LEFT_TYPE, RIGHT_TYPE, RESULT_TYPE>(
				    fun, lentry, rentry, result_validity, i);
			}
		}
	}

	template <class LEFT_TYPE, class RIGHT_TYPE, class RESULT_TYPE, class OPWRAPPER, class OP, class FUNC>
	static void ExecuteGeneric(Vector &left, Vector &right, Vector &result, idx_t count, FUNC fun) {
		UnifiedVectorFormat ldata, rdata;

		left.ToUnifiedFormat(count, ldata);
		right.ToUnifiedFormat(count, rdata);

		result.SetVectorType(VectorType::FLAT_VECTOR);
		auto result_data = FlatVector::GetData<RESULT_TYPE>(result);
		ExecuteGenericLoop<LEFT_TYPE, RIGHT_TYPE, RESULT_TYPE, OPWRAPPER, OP, FUNC>(
		    UnifiedVectorFormat::GetData<LEFT_TYPE>(ldata), UnifiedVectorFormat::GetData<RIGHT_TYPE>(rdata),
		    result_data, ldata.sel, rdata.sel, count, ldata.validity, rdata.validity, FlatVector::Validity(result),
		    fun);
	}

	template <class LEFT_TYPE, class RIGHT_TYPE, class RESULT_TYPE, class OPWRAPPER, class OP, class FUNC>
	static void ExecuteSwitch(Vector &left, Vector &right, Vector &result, idx_t count, FUNC fun) {
		auto left_vector_type = left.GetVectorType();
		auto right_vector_type = right.GetVectorType();
		if (left_vector_type == VectorType::CONSTANT_VECTOR && right_vector_type == VectorType::CONSTANT_VECTOR) {
			ExecuteConstant<LEFT_TYPE, RIGHT_TYPE, RESULT_TYPE, OPWRAPPER, OP, FUNC>(left, right, result, fun);
		} else if (left_vector_type == VectorType::FLAT_VECTOR && right_vector_type == VectorType::CONSTANT_VECTOR) {
			ExecuteFlat<LEFT_TYPE, RIGHT_TYPE, RESULT_TYPE, OPWRAPPER, OP, FUNC, false, true>(left, right, result,
			                                                                                  count, fun);
		} else if (left_vector_type == VectorType::CONSTANT_VECTOR && right_vector_type == VectorType::FLAT_VECTOR) {
			ExecuteFlat<LEFT_TYPE, RIGHT_TYPE, RESULT_TYPE, OPWRAPPER, OP, FUNC, true, false>(left, right, result,
			                                                                                  count, fun);
		} else if (left_vector_type == VectorType::FLAT_VECTOR && right_vector_type == VectorType::FLAT_VECTOR) {
			ExecuteFlat<LEFT_TYPE, RIGHT_TYPE, RESULT_TYPE, OPWRAPPER, OP, FUNC, false, false>(left, right, result,
			                                                                                   count, fun);
		} else {
			ExecuteGeneric<LEFT_TYPE, RIGHT_TYPE, RESULT_TYPE, OPWRAPPER, OP, FUNC>(left, right, result, count, fun);
		}
	}

public:
	template <class LEFT_TYPE, class RIGHT_TYPE, class RESULT_TYPE,
	          class FUNC = std::function<RESULT_TYPE(LEFT_TYPE, RIGHT_TYPE)>>
	static void Execute(Vector &left, Vector &right, Vector &result, idx_t count, FUNC fun) {
		ExecuteSwitch<LEFT_TYPE, RIGHT_TYPE, RESULT_TYPE, BinaryLambdaWrapper, bool, FUNC>(left, right, result, count,
		                                                                                   fun);
	}

	template <class LEFT_TYPE, class RIGHT_TYPE, class RESULT_TYPE, class OP,
	          class OPWRAPPER = BinarySingleArgumentOperatorWrapper>
	static void Execute(Vector &left, Vector &right, Vector &result, idx_t count) {
		ExecuteSwitch<LEFT_TYPE, RIGHT_TYPE, RESULT_TYPE, OPWRAPPER, OP, bool>(left, right, result, count, false);
	}

	template <class LEFT_TYPE, class RIGHT_TYPE, class RESULT_TYPE, class OP>
	static void ExecuteStandard(Vector &left, Vector &right, Vector &result, idx_t count) {
		ExecuteSwitch<LEFT_TYPE, RIGHT_TYPE, RESULT_TYPE, BinaryStandardOperatorWrapper, OP, bool>(left, right, result,
		                                                                                           count, false);
	}

	template <class LEFT_TYPE, class RIGHT_TYPE, class RESULT_TYPE,
	          class FUNC = std::function<RESULT_TYPE(LEFT_TYPE, RIGHT_TYPE, ValidityMask &, idx_t)>>
	static void ExecuteWithNulls(Vector &left, Vector &right, Vector &result, idx_t count, FUNC fun) {
		ExecuteSwitch<LEFT_TYPE, RIGHT_TYPE, RESULT_TYPE, BinaryLambdaWrapperWithNulls, bool, FUNC>(left, right, result,
		                                                                                            count, fun);
	}

public:
	template <class LEFT_TYPE, class RIGHT_TYPE, class OP>
	static idx_t SelectConstant(Vector &left, Vector &right, const SelectionVector *sel, idx_t count,
	                            SelectionVector *true_sel, SelectionVector *false_sel) {
		auto ldata = ConstantVector::GetData<LEFT_TYPE>(left);
		auto rdata = ConstantVector::GetData<RIGHT_TYPE>(right);

		// both sides are constant, return either 0 or the count
		// in this case we do not fill in the result selection vector at all
		if (ConstantVector::IsNull(left) || ConstantVector::IsNull(right) || !OP::Operation(*ldata, *rdata)) {
			if (false_sel) {
				for (idx_t i = 0; i < count; i++) {
					false_sel->set_index(i, sel->get_index(i));
				}
			}
			return 0;
		} else {
			if (true_sel) {
				for (idx_t i = 0; i < count; i++) {
					true_sel->set_index(i, sel->get_index(i));
				}
			}
			return count;
		}
	}

	template <class LEFT_TYPE, class RIGHT_TYPE, class OP, bool LEFT_CONSTANT, bool RIGHT_CONSTANT, bool HAS_TRUE_SEL,
	          bool HAS_FALSE_SEL>
	static inline idx_t SelectFlatLoop(const LEFT_TYPE *__restrict ldata, const RIGHT_TYPE *__restrict rdata,
	                                   const SelectionVector *sel, idx_t count, ValidityMask &validity_mask,
	                                   SelectionVector *true_sel, SelectionVector *false_sel) {
		idx_t true_count = 0, false_count = 0;
		idx_t base_idx = 0;
		auto entry_count = ValidityMask::EntryCount(count);
		for (idx_t entry_idx = 0; entry_idx < entry_count; entry_idx++) {
			auto validity_entry = validity_mask.GetValidityEntry(entry_idx);
			idx_t next = MinValue<idx_t>(base_idx + ValidityMask::BITS_PER_VALUE, count);
			if (ValidityMask::AllValid(validity_entry)) {
				// all valid: perform operation
				for (; base_idx < next; base_idx++) {
					idx_t result_idx = sel->get_index(base_idx);
					idx_t lidx = LEFT_CONSTANT ? 0 : base_idx;
					idx_t ridx = RIGHT_CONSTANT ? 0 : base_idx;
					bool comparison_result = OP::Operation(ldata[lidx], rdata[ridx]);
					if (HAS_TRUE_SEL) {
						true_sel->set_index(true_count, result_idx);
						true_count += comparison_result;
					}
					if (HAS_FALSE_SEL) {
						false_sel->set_index(false_count, result_idx);
						false_count += !comparison_result;
					}
				}
			} else if (ValidityMask::NoneValid(validity_entry)) {
				// nothing valid: skip all
				if (HAS_FALSE_SEL) {
					for (; base_idx < next; base_idx++) {
						idx_t result_idx = sel->get_index(base_idx);
						false_sel->set_index(false_count, result_idx);
						false_count++;
					}
				}
				base_idx = next;
				continue;
			} else {
				// partially valid: need to check individual elements for validity
				idx_t start = base_idx;
				for (; base_idx < next; base_idx++) {
					idx_t result_idx = sel->get_index(base_idx);
					idx_t lidx = LEFT_CONSTANT ? 0 : base_idx;
					idx_t ridx = RIGHT_CONSTANT ? 0 : base_idx;
					bool comparison_result = ValidityMask::RowIsValid(validity_entry, base_idx - start) &&
					                         OP::Operation(ldata[lidx], rdata[ridx]);
					if (HAS_TRUE_SEL) {
						true_sel->set_index(true_count, result_idx);
						true_count += comparison_result;
					}
					if (HAS_FALSE_SEL) {
						false_sel->set_index(false_count, result_idx);
						false_count += !comparison_result;
					}
				}
			}
		}
		if (HAS_TRUE_SEL) {
			return true_count;
		} else {
			return count - false_count;
		}
	}

	template <class LEFT_TYPE, class RIGHT_TYPE, class OP, bool LEFT_CONSTANT, bool RIGHT_CONSTANT>
	static inline idx_t SelectFlatLoopSwitch(const LEFT_TYPE *__restrict ldata, const RIGHT_TYPE *__restrict rdata,
	                                         const SelectionVector *sel, idx_t count, ValidityMask &mask,
	                                         SelectionVector *true_sel, SelectionVector *false_sel) {
		if (true_sel && false_sel) {
			return SelectFlatLoop<LEFT_TYPE, RIGHT_TYPE, OP, LEFT_CONSTANT, RIGHT_CONSTANT, true, true>(
			    ldata, rdata, sel, count, mask, true_sel, false_sel);
		} else if (true_sel) {
			return SelectFlatLoop<LEFT_TYPE, RIGHT_TYPE, OP, LEFT_CONSTANT, RIGHT_CONSTANT, true, false>(
			    ldata, rdata, sel, count, mask, true_sel, false_sel);
		} else {
			D_ASSERT(false_sel);
			return SelectFlatLoop<LEFT_TYPE, RIGHT_TYPE, OP, LEFT_CONSTANT, RIGHT_CONSTANT, false, true>(
			    ldata, rdata, sel, count, mask, true_sel, false_sel);
		}
	}

	template <class LEFT_TYPE, class RIGHT_TYPE, class OP, bool LEFT_CONSTANT, bool RIGHT_CONSTANT>
	static idx_t SelectFlat(Vector &left, Vector &right, const SelectionVector *sel, idx_t count,
	                        SelectionVector *true_sel, SelectionVector *false_sel) {
		auto ldata = FlatVector::GetData<LEFT_TYPE>(left);
		auto rdata = FlatVector::GetData<RIGHT_TYPE>(right);

		if (LEFT_CONSTANT && ConstantVector::IsNull(left)) {
			if (false_sel) {
				for (idx_t i = 0; i < count; i++) {
					false_sel->set_index(i, sel->get_index(i));
				}
			}
			return 0;
		}
		if (RIGHT_CONSTANT && ConstantVector::IsNull(right)) {
			if (false_sel) {
				for (idx_t i = 0; i < count; i++) {
					false_sel->set_index(i, sel->get_index(i));
				}
			}
			return 0;
		}

		if (LEFT_CONSTANT) {
			return SelectFlatLoopSwitch<LEFT_TYPE, RIGHT_TYPE, OP, LEFT_CONSTANT, RIGHT_CONSTANT>(
			    ldata, rdata, sel, count, FlatVector::Validity(right), true_sel, false_sel);
		} else if (RIGHT_CONSTANT) {
			return SelectFlatLoopSwitch<LEFT_TYPE, RIGHT_TYPE, OP, LEFT_CONSTANT, RIGHT_CONSTANT>(
			    ldata, rdata, sel, count, FlatVector::Validity(left), true_sel, false_sel);
		} else {
			ValidityMask combined_mask = FlatVector::Validity(left);
			combined_mask.Combine(FlatVector::Validity(right), count);
			return SelectFlatLoopSwitch<LEFT_TYPE, RIGHT_TYPE, OP, LEFT_CONSTANT, RIGHT_CONSTANT>(
			    ldata, rdata, sel, count, combined_mask, true_sel, false_sel);
		}
	}

	template <class LEFT_TYPE, class RIGHT_TYPE, class OP, bool NO_NULL, bool HAS_TRUE_SEL, bool HAS_FALSE_SEL>
	static inline idx_t
	SelectGenericLoop(const LEFT_TYPE *__restrict ldata, const RIGHT_TYPE *__restrict rdata,
	                  const SelectionVector *__restrict lsel, const SelectionVector *__restrict rsel,
	                  const SelectionVector *__restrict result_sel, idx_t count, ValidityMask &lvalidity,
	                  ValidityMask &rvalidity, SelectionVector *true_sel, SelectionVector *false_sel) {
		idx_t true_count = 0, false_count = 0;
		for (idx_t i = 0; i < count; i++) {
			auto result_idx = result_sel->get_index(i);
			auto lindex = lsel->get_index(i);
			auto rindex = rsel->get_index(i);
			if ((NO_NULL || (lvalidity.RowIsValid(lindex) && rvalidity.RowIsValid(rindex))) &&
			    OP::Operation(ldata[lindex], rdata[rindex])) {
				if (HAS_TRUE_SEL) {
					true_sel->set_index(true_count++, result_idx);
				}
			} else {
				if (HAS_FALSE_SEL) {
					false_sel->set_index(false_count++, result_idx);
				}
			}
		}
		if (HAS_TRUE_SEL) {
			return true_count;
		} else {
			return count - false_count;
		}
	}
	template <class LEFT_TYPE, class RIGHT_TYPE, class OP, bool NO_NULL>
	static inline idx_t
	SelectGenericLoopSelSwitch(const LEFT_TYPE *__restrict ldata, const RIGHT_TYPE *__restrict rdata,
	                           const SelectionVector *__restrict lsel, const SelectionVector *__restrict rsel,
	                           const SelectionVector *__restrict result_sel, idx_t count, ValidityMask &lvalidity,
	                           ValidityMask &rvalidity, SelectionVector *true_sel, SelectionVector *false_sel) {
		if (true_sel && false_sel) {
			return SelectGenericLoop<LEFT_TYPE, RIGHT_TYPE, OP, NO_NULL, true, true>(
			    ldata, rdata, lsel, rsel, result_sel, count, lvalidity, rvalidity, true_sel, false_sel);
		} else if (true_sel) {
			return SelectGenericLoop<LEFT_TYPE, RIGHT_TYPE, OP, NO_NULL, true, false>(
			    ldata, rdata, lsel, rsel, result_sel, count, lvalidity, rvalidity, true_sel, false_sel);
		} else {
			D_ASSERT(false_sel);
			return SelectGenericLoop<LEFT_TYPE, RIGHT_TYPE, OP, NO_NULL, false, true>(
			    ldata, rdata, lsel, rsel, result_sel, count, lvalidity, rvalidity, true_sel, false_sel);
		}
	}

	template <class LEFT_TYPE, class RIGHT_TYPE, class OP>
	static inline idx_t
	SelectGenericLoopSwitch(const LEFT_TYPE *__restrict ldata, const RIGHT_TYPE *__restrict rdata,
	                        const SelectionVector *__restrict lsel, const SelectionVector *__restrict rsel,
	                        const SelectionVector *__restrict result_sel, idx_t count, ValidityMask &lvalidity,
	                        ValidityMask &rvalidity, SelectionVector *true_sel, SelectionVector *false_sel) {
		if (!lvalidity.AllValid() || !rvalidity.AllValid()) {
			return SelectGenericLoopSelSwitch<LEFT_TYPE, RIGHT_TYPE, OP, false>(
			    ldata, rdata, lsel, rsel, result_sel, count, lvalidity, rvalidity, true_sel, false_sel);
		} else {
			return SelectGenericLoopSelSwitch<LEFT_TYPE, RIGHT_TYPE, OP, true>(
			    ldata, rdata, lsel, rsel, result_sel, count, lvalidity, rvalidity, true_sel, false_sel);
		}
	}

	template <class LEFT_TYPE, class RIGHT_TYPE, class OP>
	static idx_t SelectGeneric(Vector &left, Vector &right, const SelectionVector *sel, idx_t count,
	                           SelectionVector *true_sel, SelectionVector *false_sel) {
		UnifiedVectorFormat ldata, rdata;

		left.ToUnifiedFormat(count, ldata);
		right.ToUnifiedFormat(count, rdata);

		return SelectGenericLoopSwitch<LEFT_TYPE, RIGHT_TYPE, OP>(
		    UnifiedVectorFormat::GetData<LEFT_TYPE>(ldata), UnifiedVectorFormat::GetData<RIGHT_TYPE>(rdata), ldata.sel,
		    rdata.sel, sel, count, ldata.validity, rdata.validity, true_sel, false_sel);
	}

	template <class LEFT_TYPE, class RIGHT_TYPE, class OP>
	static idx_t Select(Vector &left, Vector &right, const SelectionVector *sel, idx_t count, SelectionVector *true_sel,
	                    SelectionVector *false_sel) {
		if (!sel) {
			sel = FlatVector::IncrementalSelectionVector();
		}
		if (left.GetVectorType() == VectorType::CONSTANT_VECTOR &&
		    right.GetVectorType() == VectorType::CONSTANT_VECTOR) {
			return SelectConstant<LEFT_TYPE, RIGHT_TYPE, OP>(left, right, sel, count, true_sel, false_sel);
		} else if (left.GetVectorType() == VectorType::CONSTANT_VECTOR &&
		           right.GetVectorType() == VectorType::FLAT_VECTOR) {
			return SelectFlat<LEFT_TYPE, RIGHT_TYPE, OP, true, false>(left, right, sel, count, true_sel, false_sel);
		} else if (left.GetVectorType() == VectorType::FLAT_VECTOR &&
		           right.GetVectorType() == VectorType::CONSTANT_VECTOR) {
			return SelectFlat<LEFT_TYPE, RIGHT_TYPE, OP, false, true>(left, right, sel, count, true_sel, false_sel);
		} else if (left.GetVectorType() == VectorType::FLAT_VECTOR &&
		           right.GetVectorType() == VectorType::FLAT_VECTOR) {
			return SelectFlat<LEFT_TYPE, RIGHT_TYPE, OP, false, false>(left, right, sel, count, true_sel, false_sel);
		} else {
			return SelectGeneric<LEFT_TYPE, RIGHT_TYPE, OP>(left, right, sel, count, true_sel, false_sel);
		}
	}
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/vector_operations/ternary_executor.hpp
//
//
//===----------------------------------------------------------------------===//







#include <functional>

namespace duckdb {

template <class OP>
struct TernaryStandardOperatorWrapper {
	template <class FUN, class A_TYPE, class B_TYPE, class C_TYPE, class RESULT_TYPE>
	static inline RESULT_TYPE Operation(FUN fun, A_TYPE a, B_TYPE b, C_TYPE c, ValidityMask &mask, idx_t idx) {
		return OP::template Operation<A_TYPE, B_TYPE, C_TYPE, RESULT_TYPE>(a, b, c);
	}
};

struct TernaryLambdaWrapper {
	template <class FUN, class A_TYPE, class B_TYPE, class C_TYPE, class RESULT_TYPE>
	static inline RESULT_TYPE Operation(FUN fun, A_TYPE a, B_TYPE b, C_TYPE c, ValidityMask &mask, idx_t idx) {
		return fun(a, b, c);
	}
};

struct TernaryLambdaWrapperWithNulls {
	template <class FUN, class A_TYPE, class B_TYPE, class C_TYPE, class RESULT_TYPE>
	static inline RESULT_TYPE Operation(FUN fun, A_TYPE a, B_TYPE b, C_TYPE c, ValidityMask &mask, idx_t idx) {
		return fun(a, b, c, mask, idx);
	}
};

struct TernaryExecutor {
private:
	template <class A_TYPE, class B_TYPE, class C_TYPE, class RESULT_TYPE, class OPWRAPPER, class FUN>
	static inline void ExecuteLoop(const A_TYPE *__restrict adata, const B_TYPE *__restrict bdata,
	                               const C_TYPE *__restrict cdata, RESULT_TYPE *__restrict result_data, idx_t count,
	                               const SelectionVector &asel, const SelectionVector &bsel,
	                               const SelectionVector &csel, ValidityMask &avalidity, ValidityMask &bvalidity,
	                               ValidityMask &cvalidity, ValidityMask &result_validity, FUN fun) {
		if (!avalidity.AllValid() || !bvalidity.AllValid() || !cvalidity.AllValid()) {
			for (idx_t i = 0; i < count; i++) {
				auto aidx = asel.get_index(i);
				auto bidx = bsel.get_index(i);
				auto cidx = csel.get_index(i);
				if (avalidity.RowIsValid(aidx) && bvalidity.RowIsValid(bidx) && cvalidity.RowIsValid(cidx)) {
					result_data[i] = OPWRAPPER::template Operation<FUN, A_TYPE, B_TYPE, C_TYPE, RESULT_TYPE>(
					    fun, adata[aidx], bdata[bidx], cdata[cidx], result_validity, i);
				} else {
					result_validity.SetInvalid(i);
				}
			}
		} else {
			for (idx_t i = 0; i < count; i++) {
				auto aidx = asel.get_index(i);
				auto bidx = bsel.get_index(i);
				auto cidx = csel.get_index(i);
				result_data[i] = OPWRAPPER::template Operation<FUN, A_TYPE, B_TYPE, C_TYPE, RESULT_TYPE>(
				    fun, adata[aidx], bdata[bidx], cdata[cidx], result_validity, i);
			}
		}
	}

public:
	template <class A_TYPE, class B_TYPE, class C_TYPE, class RESULT_TYPE, class OPWRAPPER, class FUN>
	static void ExecuteGeneric(Vector &a, Vector &b, Vector &c, Vector &result, idx_t count, FUN fun) {
		if (a.GetVectorType() == VectorType::CONSTANT_VECTOR && b.GetVectorType() == VectorType::CONSTANT_VECTOR &&
		    c.GetVectorType() == VectorType::CONSTANT_VECTOR) {
			result.SetVectorType(VectorType::CONSTANT_VECTOR);
			if (ConstantVector::IsNull(a) || ConstantVector::IsNull(b) || ConstantVector::IsNull(c)) {
				ConstantVector::SetNull(result, true);
			} else {
				auto adata = ConstantVector::GetData<A_TYPE>(a);
				auto bdata = ConstantVector::GetData<B_TYPE>(b);
				auto cdata = ConstantVector::GetData<C_TYPE>(c);
				auto result_data = ConstantVector::GetData<RESULT_TYPE>(result);
				auto &result_validity = ConstantVector::Validity(result);
				result_data[0] = OPWRAPPER::template Operation<FUN, A_TYPE, B_TYPE, C_TYPE, RESULT_TYPE>(
				    fun, adata[0], bdata[0], cdata[0], result_validity, 0);
			}
		} else {
			result.SetVectorType(VectorType::FLAT_VECTOR);

			UnifiedVectorFormat adata, bdata, cdata;
			a.ToUnifiedFormat(count, adata);
			b.ToUnifiedFormat(count, bdata);
			c.ToUnifiedFormat(count, cdata);

			ExecuteLoop<A_TYPE, B_TYPE, C_TYPE, RESULT_TYPE, OPWRAPPER>(
			    UnifiedVectorFormat::GetData<A_TYPE>(adata), UnifiedVectorFormat::GetData<B_TYPE>(bdata),
			    UnifiedVectorFormat::GetData<C_TYPE>(cdata), FlatVector::GetData<RESULT_TYPE>(result), count,
			    *adata.sel, *bdata.sel, *cdata.sel, adata.validity, bdata.validity, cdata.validity,
			    FlatVector::Validity(result), fun);
		}
	}

	template <class A_TYPE, class B_TYPE, class C_TYPE, class RESULT_TYPE,
	          class FUN = std::function<RESULT_TYPE(A_TYPE, B_TYPE, C_TYPE)>>
	static void Execute(Vector &a, Vector &b, Vector &c, Vector &result, idx_t count, FUN fun) {
		ExecuteGeneric<A_TYPE, B_TYPE, C_TYPE, RESULT_TYPE, TernaryLambdaWrapper, FUN>(a, b, c, result, count, fun);
	}

	template <class A_TYPE, class B_TYPE, class C_TYPE, class RESULT_TYPE, class OP>
	static void ExecuteStandard(Vector &a, Vector &b, Vector &c, Vector &result, idx_t count) {
		ExecuteGeneric<A_TYPE, B_TYPE, C_TYPE, RESULT_TYPE, TernaryStandardOperatorWrapper<OP>, bool>(a, b, c, result,
		                                                                                              count, false);
	}

	template <class A_TYPE, class B_TYPE, class C_TYPE, class RESULT_TYPE,
	          class FUN = std::function<RESULT_TYPE(A_TYPE, B_TYPE, C_TYPE, ValidityMask &, idx_t)>>
	static void ExecuteWithNulls(Vector &a, Vector &b, Vector &c, Vector &result, idx_t count, FUN fun) {
		ExecuteGeneric<A_TYPE, B_TYPE, C_TYPE, RESULT_TYPE, TernaryLambdaWrapperWithNulls, FUN>(a, b, c, result, count,
		                                                                                        fun);
	}

private:
	template <class A_TYPE, class B_TYPE, class C_TYPE, class OP, bool NO_NULL, bool HAS_TRUE_SEL, bool HAS_FALSE_SEL>
	static inline idx_t SelectLoop(const A_TYPE *__restrict adata, const B_TYPE *__restrict bdata,
	                               const C_TYPE *__restrict cdata, const SelectionVector *result_sel, idx_t count,
	                               const SelectionVector &asel, const SelectionVector &bsel,
	                               const SelectionVector &csel, ValidityMask &avalidity, ValidityMask &bvalidity,
	                               ValidityMask &cvalidity, SelectionVector *true_sel, SelectionVector *false_sel) {
		idx_t true_count = 0, false_count = 0;
		for (idx_t i = 0; i < count; i++) {
			auto result_idx = result_sel->get_index(i);
			auto aidx = asel.get_index(i);
			auto bidx = bsel.get_index(i);
			auto cidx = csel.get_index(i);
			bool comparison_result =
			    (NO_NULL || (avalidity.RowIsValid(aidx) && bvalidity.RowIsValid(bidx) && cvalidity.RowIsValid(cidx))) &&
			    OP::Operation(adata[aidx], bdata[bidx], cdata[cidx]);
			if (HAS_TRUE_SEL) {
				true_sel->set_index(true_count, result_idx);
				true_count += comparison_result;
			}
			if (HAS_FALSE_SEL) {
				false_sel->set_index(false_count, result_idx);
				false_count += !comparison_result;
			}
		}
		if (HAS_TRUE_SEL) {
			return true_count;
		} else {
			return count - false_count;
		}
	}

	template <class A_TYPE, class B_TYPE, class C_TYPE, class OP, bool NO_NULL>
	static inline idx_t SelectLoopSelSwitch(UnifiedVectorFormat &adata, UnifiedVectorFormat &bdata,
	                                        UnifiedVectorFormat &cdata, const SelectionVector *sel, idx_t count,
	                                        SelectionVector *true_sel, SelectionVector *false_sel) {
		if (true_sel && false_sel) {
			return SelectLoop<A_TYPE, B_TYPE, C_TYPE, OP, NO_NULL, true, true>(
			    UnifiedVectorFormat::GetData<A_TYPE>(adata), UnifiedVectorFormat::GetData<B_TYPE>(bdata),
			    UnifiedVectorFormat::GetData<C_TYPE>(cdata), sel, count, *adata.sel, *bdata.sel, *cdata.sel,
			    adata.validity, bdata.validity, cdata.validity, true_sel, false_sel);
		} else if (true_sel) {
			return SelectLoop<A_TYPE, B_TYPE, C_TYPE, OP, NO_NULL, true, false>(
			    UnifiedVectorFormat::GetData<A_TYPE>(adata), UnifiedVectorFormat::GetData<B_TYPE>(bdata),
			    UnifiedVectorFormat::GetData<C_TYPE>(cdata), sel, count, *adata.sel, *bdata.sel, *cdata.sel,
			    adata.validity, bdata.validity, cdata.validity, true_sel, false_sel);
		} else {
			D_ASSERT(false_sel);
			return SelectLoop<A_TYPE, B_TYPE, C_TYPE, OP, NO_NULL, false, true>(
			    UnifiedVectorFormat::GetData<A_TYPE>(adata), UnifiedVectorFormat::GetData<B_TYPE>(bdata),
			    UnifiedVectorFormat::GetData<C_TYPE>(cdata), sel, count, *adata.sel, *bdata.sel, *cdata.sel,
			    adata.validity, bdata.validity, cdata.validity, true_sel, false_sel);
		}
	}

	template <class A_TYPE, class B_TYPE, class C_TYPE, class OP>
	static inline idx_t SelectLoopSwitch(UnifiedVectorFormat &adata, UnifiedVectorFormat &bdata,
	                                     UnifiedVectorFormat &cdata, const SelectionVector *sel, idx_t count,
	                                     SelectionVector *true_sel, SelectionVector *false_sel) {
		if (!adata.validity.AllValid() || !bdata.validity.AllValid() || !cdata.validity.AllValid()) {
			return SelectLoopSelSwitch<A_TYPE, B_TYPE, C_TYPE, OP, false>(adata, bdata, cdata, sel, count, true_sel,
			                                                              false_sel);
		} else {
			return SelectLoopSelSwitch<A_TYPE, B_TYPE, C_TYPE, OP, true>(adata, bdata, cdata, sel, count, true_sel,
			                                                             false_sel);
		}
	}

public:
	template <class A_TYPE, class B_TYPE, class C_TYPE, class OP>
	static idx_t Select(Vector &a, Vector &b, Vector &c, const SelectionVector *sel, idx_t count,
	                    SelectionVector *true_sel, SelectionVector *false_sel) {
		if (!sel) {
			sel = FlatVector::IncrementalSelectionVector();
		}
		UnifiedVectorFormat adata, bdata, cdata;
		a.ToUnifiedFormat(count, adata);
		b.ToUnifiedFormat(count, bdata);
		c.ToUnifiedFormat(count, cdata);

		return SelectLoopSwitch<A_TYPE, B_TYPE, C_TYPE, OP>(adata, bdata, cdata, sel, count, true_sel, false_sel);
	}
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/vector_operations/unary_executor.hpp
//
//
//===----------------------------------------------------------------------===//







#include <functional>

namespace duckdb {

struct UnaryOperatorWrapper {
	template <class OP, class INPUT_TYPE, class RESULT_TYPE>
	static inline RESULT_TYPE Operation(INPUT_TYPE input, ValidityMask &mask, idx_t idx, void *dataptr) {
		return OP::template Operation<INPUT_TYPE, RESULT_TYPE>(input);
	}
};

struct UnaryLambdaWrapper {
	template <class FUNC, class INPUT_TYPE, class RESULT_TYPE>
	static inline RESULT_TYPE Operation(INPUT_TYPE input, ValidityMask &mask, idx_t idx, void *dataptr) {
		auto fun = (FUNC *)dataptr;
		return (*fun)(input);
	}
};

struct GenericUnaryWrapper {
	template <class OP, class INPUT_TYPE, class RESULT_TYPE>
	static inline RESULT_TYPE Operation(INPUT_TYPE input, ValidityMask &mask, idx_t idx, void *dataptr) {
		return OP::template Operation<INPUT_TYPE, RESULT_TYPE>(input, mask, idx, dataptr);
	}
};

struct UnaryLambdaWrapperWithNulls {
	template <class FUNC, class INPUT_TYPE, class RESULT_TYPE>
	static inline RESULT_TYPE Operation(INPUT_TYPE input, ValidityMask &mask, idx_t idx, void *dataptr) {
		auto fun = (FUNC *)dataptr;
		return (*fun)(input, mask, idx);
	}
};

template <class OP>
struct UnaryStringOperator {
	template <class INPUT_TYPE, class RESULT_TYPE>
	static RESULT_TYPE Operation(INPUT_TYPE input, ValidityMask &mask, idx_t idx, void *dataptr) {
		auto vector = (Vector *)dataptr;
		return OP::template Operation<INPUT_TYPE, RESULT_TYPE>(input, *vector);
	}
};

struct UnaryExecutor {
private:
	template <class INPUT_TYPE, class RESULT_TYPE, class OPWRAPPER, class OP>
	static inline void ExecuteLoop(const INPUT_TYPE *__restrict ldata, RESULT_TYPE *__restrict result_data, idx_t count,
	                               const SelectionVector *__restrict sel_vector, ValidityMask &mask,
	                               ValidityMask &result_mask, void *dataptr, bool adds_nulls) {
#ifdef DEBUG
		// ldata may point to a compressed dictionary buffer which can be smaller than ldata + count
		idx_t max_index = 0;
		for (idx_t i = 0; i < count; i++) {
			auto idx = sel_vector->get_index(i);
			max_index = MaxValue(max_index, idx);
		}
		ASSERT_RESTRICT(ldata, ldata + max_index, result_data, result_data + count);
#endif

		if (!mask.AllValid()) {
			result_mask.EnsureWritable();
			for (idx_t i = 0; i < count; i++) {
				auto idx = sel_vector->get_index(i);
				if (mask.RowIsValidUnsafe(idx)) {
					result_data[i] =
					    OPWRAPPER::template Operation<OP, INPUT_TYPE, RESULT_TYPE>(ldata[idx], result_mask, i, dataptr);
				} else {
					result_mask.SetInvalid(i);
				}
			}
		} else {
			if (adds_nulls) {
				result_mask.EnsureWritable();
			}
			for (idx_t i = 0; i < count; i++) {
				auto idx = sel_vector->get_index(i);
				result_data[i] =
				    OPWRAPPER::template Operation<OP, INPUT_TYPE, RESULT_TYPE>(ldata[idx], result_mask, i, dataptr);
			}
		}
	}

	template <class INPUT_TYPE, class RESULT_TYPE, class OPWRAPPER, class OP>
	static inline void ExecuteFlat(const INPUT_TYPE *__restrict ldata, RESULT_TYPE *__restrict result_data, idx_t count,
	                               ValidityMask &mask, ValidityMask &result_mask, void *dataptr, bool adds_nulls) {
		ASSERT_RESTRICT(ldata, ldata + count, result_data, result_data + count);

		if (!mask.AllValid()) {
			if (!adds_nulls) {
				result_mask.Initialize(mask);
			} else {
				result_mask.Copy(mask, count);
			}
			idx_t base_idx = 0;
			auto entry_count = ValidityMask::EntryCount(count);
			for (idx_t entry_idx = 0; entry_idx < entry_count; entry_idx++) {
				auto validity_entry = mask.GetValidityEntry(entry_idx);
				idx_t next = MinValue<idx_t>(base_idx + ValidityMask::BITS_PER_VALUE, count);
				if (ValidityMask::AllValid(validity_entry)) {
					// all valid: perform operation
					for (; base_idx < next; base_idx++) {
						result_data[base_idx] = OPWRAPPER::template Operation<OP, INPUT_TYPE, RESULT_TYPE>(
						    ldata[base_idx], result_mask, base_idx, dataptr);
					}
				} else if (ValidityMask::NoneValid(validity_entry)) {
					// nothing valid: skip all
					base_idx = next;
					continue;
				} else {
					// partially valid: need to check individual elements for validity
					idx_t start = base_idx;
					for (; base_idx < next; base_idx++) {
						if (ValidityMask::RowIsValid(validity_entry, base_idx - start)) {
							D_ASSERT(mask.RowIsValid(base_idx));
							result_data[base_idx] = OPWRAPPER::template Operation<OP, INPUT_TYPE, RESULT_TYPE>(
							    ldata[base_idx], result_mask, base_idx, dataptr);
						}
					}
				}
			}
		} else {
			if (adds_nulls) {
				result_mask.EnsureWritable();
			}
			for (idx_t i = 0; i < count; i++) {
				result_data[i] =
				    OPWRAPPER::template Operation<OP, INPUT_TYPE, RESULT_TYPE>(ldata[i], result_mask, i, dataptr);
			}
		}
	}

	template <class INPUT_TYPE, class RESULT_TYPE, class OPWRAPPER, class OP>
	static inline void ExecuteStandard(Vector &input, Vector &result, idx_t count, void *dataptr, bool adds_nulls) {
		switch (input.GetVectorType()) {
		case VectorType::CONSTANT_VECTOR: {
			result.SetVectorType(VectorType::CONSTANT_VECTOR);
			auto result_data = ConstantVector::GetData<RESULT_TYPE>(result);
			auto ldata = ConstantVector::GetData<INPUT_TYPE>(input);

			if (ConstantVector::IsNull(input)) {
				ConstantVector::SetNull(result, true);
			} else {
				ConstantVector::SetNull(result, false);
				*result_data = OPWRAPPER::template Operation<OP, INPUT_TYPE, RESULT_TYPE>(
				    *ldata, ConstantVector::Validity(result), 0, dataptr);
			}
			break;
		}
		case VectorType::FLAT_VECTOR: {
			result.SetVectorType(VectorType::FLAT_VECTOR);
			auto result_data = FlatVector::GetData<RESULT_TYPE>(result);
			auto ldata = FlatVector::GetData<INPUT_TYPE>(input);

			ExecuteFlat<INPUT_TYPE, RESULT_TYPE, OPWRAPPER, OP>(ldata, result_data, count, FlatVector::Validity(input),
			                                                    FlatVector::Validity(result), dataptr, adds_nulls);
			break;
		}
		default: {
			UnifiedVectorFormat vdata;
			input.ToUnifiedFormat(count, vdata);

			result.SetVectorType(VectorType::FLAT_VECTOR);
			auto result_data = FlatVector::GetData<RESULT_TYPE>(result);
			auto ldata = UnifiedVectorFormat::GetData<INPUT_TYPE>(vdata);

			ExecuteLoop<INPUT_TYPE, RESULT_TYPE, OPWRAPPER, OP>(ldata, result_data, count, vdata.sel, vdata.validity,
			                                                    FlatVector::Validity(result), dataptr, adds_nulls);
			break;
		}
		}
	}

public:
	template <class INPUT_TYPE, class RESULT_TYPE, class OP>
	static void Execute(Vector &input, Vector &result, idx_t count) {
		ExecuteStandard<INPUT_TYPE, RESULT_TYPE, UnaryOperatorWrapper, OP>(input, result, count, nullptr, false);
	}

	template <class INPUT_TYPE, class RESULT_TYPE, class FUNC = std::function<RESULT_TYPE(INPUT_TYPE)>>
	static void Execute(Vector &input, Vector &result, idx_t count, FUNC fun) {
		ExecuteStandard<INPUT_TYPE, RESULT_TYPE, UnaryLambdaWrapper, FUNC>(input, result, count, (void *)&fun, false);
	}

	template <class INPUT_TYPE, class RESULT_TYPE, class OP>
	static void GenericExecute(Vector &input, Vector &result, idx_t count, void *dataptr, bool adds_nulls = false) {
		ExecuteStandard<INPUT_TYPE, RESULT_TYPE, GenericUnaryWrapper, OP>(input, result, count, dataptr, adds_nulls);
	}

	template <class INPUT_TYPE, class RESULT_TYPE,
	          class FUNC = std::function<RESULT_TYPE(INPUT_TYPE, ValidityMask &, idx_t)>>
	static void ExecuteWithNulls(Vector &input, Vector &result, idx_t count, FUNC fun) {
		ExecuteStandard<INPUT_TYPE, RESULT_TYPE, UnaryLambdaWrapperWithNulls, FUNC>(input, result, count, (void *)&fun,
		                                                                            true);
	}

	template <class INPUT_TYPE, class RESULT_TYPE, class OP>
	static void ExecuteString(Vector &input, Vector &result, idx_t count) {
		UnaryExecutor::GenericExecute<INPUT_TYPE, RESULT_TYPE, UnaryStringOperator<OP>>(input, result, count,
		                                                                                (void *)&result);
	}
};

} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/execution/expression_executor_state.hpp
//
//
//===----------------------------------------------------------------------===//





//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/cycle_counter.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/chrono.hpp
//
//
//===----------------------------------------------------------------------===//



#include <chrono>

namespace duckdb {
using std::chrono::duration;
using std::chrono::duration_cast;
using std::chrono::high_resolution_clock;
using std::chrono::milliseconds;
using std::chrono::system_clock;
using std::chrono::time_point;
} // namespace duckdb


namespace duckdb {

//! The cycle counter can be used to measure elapsed cycles for a function, expression and ...
//! Optimized by sampling mechanism. Once per 100 times.
//! //Todo Can be optimized further by calling RDTSC once per sample
class CycleCounter {
	friend struct ExpressionInfo;
	friend struct ExpressionRootInfo;
	static constexpr int SAMPLING_RATE = 50;

public:
	CycleCounter() {
	}
	// Next_sample determines if a sample needs to be taken, if so start the profiler
	void BeginSample() {
		if (current_count >= next_sample) {
			tmp = Tick();
		}
	}

	// End the sample
	void EndSample(int chunk_size) {
		if (current_count >= next_sample) {
			time += Tick() - tmp;
		}
		if (current_count >= next_sample) {
			next_sample = SAMPLING_RATE;
			++sample_count;
			sample_tuples_count += chunk_size;
			current_count = 0;
		} else {
			++current_count;
		}
		tuples_count += chunk_size;
	}

private:
	uint64_t Tick() const;
	// current number on RDT register
	uint64_t tmp;
	// Elapsed cycles
	uint64_t time = 0;
	//! Count the number of time the executor called since last sampling
	uint64_t current_count = 0;
	//! Show the next sample
	uint64_t next_sample = 0;
	//! Count the number of samples
	uint64_t sample_count = 0;
	//! Count the number of tuples sampled
	uint64_t sample_tuples_count = 0;
	//! Count the number of ALL tuples
	uint64_t tuples_count = 0;
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/function/function.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/named_parameter_map.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/case_insensitive_map.hpp
//
//
//===----------------------------------------------------------------------===//








namespace duckdb {

struct CaseInsensitiveStringHashFunction {
	uint64_t operator()(const string &str) const {
		return StringUtil::CIHash(str);
	}
};

struct CaseInsensitiveStringEquality {
	bool operator()(const string &a, const string &b) const {
		return StringUtil::CIEquals(a, b);
	}
};

template <typename T>
using case_insensitive_map_t =
    unordered_map<string, T, CaseInsensitiveStringHashFunction, CaseInsensitiveStringEquality>;

using case_insensitive_set_t = unordered_set<string, CaseInsensitiveStringHashFunction, CaseInsensitiveStringEquality>;

} // namespace duckdb


namespace duckdb {

using named_parameter_type_map_t = case_insensitive_map_t<LogicalType>;
using named_parameter_map_t = case_insensitive_map_t<Value>;

} // namespace duckdb



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/main/external_dependencies.hpp
//
//
//===----------------------------------------------------------------------===//



namespace duckdb {

enum ExternalDependenciesType { PYTHON_DEPENDENCY };
class ExternalDependency {
public:
	explicit ExternalDependency(ExternalDependenciesType type_p) : type(type_p) {};
	virtual ~ExternalDependency() {};
	ExternalDependenciesType type;
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/column_definition.hpp
//
//
//===----------------------------------------------------------------------===//





//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/parsed_expression.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/base_expression.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/expression_type.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

//===--------------------------------------------------------------------===//
// Predicate Expression Operation Types
//===--------------------------------------------------------------------===//
enum class ExpressionType : uint8_t {
	INVALID = 0,

	// explicitly cast left as right (right is integer in ValueType enum)
	OPERATOR_CAST = 12,
	// logical not operator
	OPERATOR_NOT = 13,
	// is null operator
	OPERATOR_IS_NULL = 14,
	// is not null operator
	OPERATOR_IS_NOT_NULL = 15,

	// -----------------------------
	// Comparison Operators
	// -----------------------------
	// equal operator between left and right
	COMPARE_EQUAL = 25,
	// compare initial boundary
	COMPARE_BOUNDARY_START = COMPARE_EQUAL,
	// inequal operator between left and right
	COMPARE_NOTEQUAL = 26,
	// less than operator between left and right
	COMPARE_LESSTHAN = 27,
	// greater than operator between left and right
	COMPARE_GREATERTHAN = 28,
	// less than equal operator between left and right
	COMPARE_LESSTHANOREQUALTO = 29,
	// greater than equal operator between left and right
	COMPARE_GREATERTHANOREQUALTO = 30,
	// IN operator [left IN (right1, right2, ...)]
	COMPARE_IN = 35,
	// NOT IN operator [left NOT IN (right1, right2, ...)]
	COMPARE_NOT_IN = 36,
	// IS DISTINCT FROM operator
	COMPARE_DISTINCT_FROM = 37,

	COMPARE_BETWEEN = 38,
	COMPARE_NOT_BETWEEN = 39,
	// IS NOT DISTINCT FROM operator
	COMPARE_NOT_DISTINCT_FROM = 40,
	// compare final boundary
	COMPARE_BOUNDARY_END = COMPARE_NOT_DISTINCT_FROM,

	// -----------------------------
	// Conjunction Operators
	// -----------------------------
	CONJUNCTION_AND = 50,
	CONJUNCTION_OR = 51,

	// -----------------------------
	// Values
	// -----------------------------
	VALUE_CONSTANT = 75,
	VALUE_PARAMETER = 76,
	VALUE_TUPLE = 77,
	VALUE_TUPLE_ADDRESS = 78,
	VALUE_NULL = 79,
	VALUE_VECTOR = 80,
	VALUE_SCALAR = 81,
	VALUE_DEFAULT = 82,

	// -----------------------------
	// Aggregates
	// -----------------------------
	AGGREGATE = 100,
	BOUND_AGGREGATE = 101,
	GROUPING_FUNCTION = 102,

	// -----------------------------
	// Window Functions
	// -----------------------------
	WINDOW_AGGREGATE = 110,

	WINDOW_RANK = 120,
	WINDOW_RANK_DENSE = 121,
	WINDOW_NTILE = 122,
	WINDOW_PERCENT_RANK = 123,
	WINDOW_CUME_DIST = 124,
	WINDOW_ROW_NUMBER = 125,

	WINDOW_FIRST_VALUE = 130,
	WINDOW_LAST_VALUE = 131,
	WINDOW_LEAD = 132,
	WINDOW_LAG = 133,
	WINDOW_NTH_VALUE = 134,

	// -----------------------------
	// Functions
	// -----------------------------
	FUNCTION = 140,
	BOUND_FUNCTION = 141,

	// -----------------------------
	// Operators
	// -----------------------------
	CASE_EXPR = 150,
	OPERATOR_NULLIF = 151,
	OPERATOR_COALESCE = 152,
	ARRAY_EXTRACT = 153,
	ARRAY_SLICE = 154,
	STRUCT_EXTRACT = 155,
	ARRAY_CONSTRUCTOR = 156,
	ARROW = 157,

	// -----------------------------
	// Subquery IN/EXISTS
	// -----------------------------
	SUBQUERY = 175,

	// -----------------------------
	// Parser
	// -----------------------------
	STAR = 200,
	TABLE_STAR = 201,
	PLACEHOLDER = 202,
	COLUMN_REF = 203,
	FUNCTION_REF = 204,
	TABLE_REF = 205,

	// -----------------------------
	// Miscellaneous
	// -----------------------------
	CAST = 225,
	BOUND_REF = 227,
	BOUND_COLUMN_REF = 228,
	BOUND_UNNEST = 229,
	COLLATE = 230,
	LAMBDA = 231,
	POSITIONAL_REFERENCE = 232,
	BOUND_LAMBDA_REF = 233
};

//===--------------------------------------------------------------------===//
// Expression Class
//===--------------------------------------------------------------------===//
enum class ExpressionClass : uint8_t {
	INVALID = 0,
	//===--------------------------------------------------------------------===//
	// Parsed Expressions
	//===--------------------------------------------------------------------===//
	AGGREGATE = 1,
	CASE = 2,
	CAST = 3,
	COLUMN_REF = 4,
	COMPARISON = 5,
	CONJUNCTION = 6,
	CONSTANT = 7,
	DEFAULT = 8,
	FUNCTION = 9,
	OPERATOR = 10,
	STAR = 11,
	SUBQUERY = 13,
	WINDOW = 14,
	PARAMETER = 15,
	COLLATE = 16,
	LAMBDA = 17,
	POSITIONAL_REFERENCE = 18,
	BETWEEN = 19,
	//===--------------------------------------------------------------------===//
	// Bound Expressions
	//===--------------------------------------------------------------------===//
	BOUND_AGGREGATE = 25,
	BOUND_CASE = 26,
	BOUND_CAST = 27,
	BOUND_COLUMN_REF = 28,
	BOUND_COMPARISON = 29,
	BOUND_CONJUNCTION = 30,
	BOUND_CONSTANT = 31,
	BOUND_DEFAULT = 32,
	BOUND_FUNCTION = 33,
	BOUND_OPERATOR = 34,
	BOUND_PARAMETER = 35,
	BOUND_REF = 36,
	BOUND_SUBQUERY = 37,
	BOUND_WINDOW = 38,
	BOUND_BETWEEN = 39,
	BOUND_UNNEST = 40,
	BOUND_LAMBDA = 41,
	BOUND_LAMBDA_REF = 42,
	//===--------------------------------------------------------------------===//
	// Miscellaneous
	//===--------------------------------------------------------------------===//
	BOUND_EXPRESSION = 50
};

DUCKDB_API string ExpressionTypeToString(ExpressionType type);
string ExpressionTypeToOperator(ExpressionType type);

// Operator String to ExpressionType (e.g. + => OPERATOR_ADD)
ExpressionType OperatorToExpressionType(const string &op);
//! Negate a comparison expression, turning e.g. = into !=, or < into >=
ExpressionType NegateComparisonExpression(ExpressionType type);
//! Flip a comparison expression, turning e.g. < into >, or = into =
ExpressionType FlipComparisonExpression(ExpressionType type);

DUCKDB_API string ExpressionClassToString(ExpressionClass type);

} // namespace duckdb



namespace duckdb {

//!  The BaseExpression class is a base class that can represent any expression
//!  part of a SQL statement.
class BaseExpression {
public:
	//! Create an Expression
	BaseExpression(ExpressionType type, ExpressionClass expression_class)
	    : type(type), expression_class(expression_class) {
	}
	virtual ~BaseExpression() {
	}

	//! Returns the type of the expression
	ExpressionType GetExpressionType() const {
		return type;
	}
	//! Returns the class of the expression
	ExpressionClass GetExpressionClass() const {
		return expression_class;
	}

	//! Type of the expression
	ExpressionType type;
	//! The expression class of the node
	ExpressionClass expression_class;
	//! The alias of the expression,
	string alias;

public:
	//! Returns true if this expression is an aggregate or not.
	/*!
	 Examples:

	 (1) SUM(a) + 1 -- True

	 (2) a + 1 -- False
	 */
	virtual bool IsAggregate() const = 0;
	//! Returns true if the expression has a window function or not
	virtual bool IsWindow() const = 0;
	//! Returns true if the query contains a subquery
	virtual bool HasSubquery() const = 0;
	//! Returns true if expression does not contain a group ref or col ref or parameter
	virtual bool IsScalar() const = 0;
	//! Returns true if the expression has a parameter
	virtual bool HasParameter() const = 0;

	//! Get the name of the expression
	virtual string GetName() const;
	//! Convert the Expression to a String
	virtual string ToString() const = 0;
	//! Print the expression to stdout
	void Print() const;

	//! Creates a hash value of this expression. It is important that if two expressions are identical (i.e.
	//! Expression::Equals() returns true), that their hash value is identical as well.
	virtual hash_t Hash() const = 0;
	//! Returns true if this expression is equal to another expression
	virtual bool Equals(const BaseExpression &other) const;

	static bool Equals(const BaseExpression &left, const BaseExpression &right) {
		return left.Equals(right);
	}
	bool operator==(const BaseExpression &rhs) {
		return Equals(rhs);
	}

	virtual void Verify() const;

public:
	template <class TARGET>
	TARGET &Cast() {
		if (expression_class != TARGET::TYPE) {
			throw InternalException("Failed to cast expression to type - expression type mismatch");
		}
		return reinterpret_cast<TARGET &>(*this);
	}

	template <class TARGET>
	const TARGET &Cast() const {
		if (expression_class != TARGET::TYPE) {
			throw InternalException("Failed to cast expression to type - expression type mismatch");
		}
		return reinterpret_cast<const TARGET &>(*this);
	}
};

} // namespace duckdb



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/qualified_name.hpp
//
//
//===----------------------------------------------------------------------===//





//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/keyword_helper.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

class KeywordHelper {
public:
	//! Returns true if the given text matches a keyword of the parser
	static bool IsKeyword(const string &text);

	static string EscapeQuotes(const string &text, char quote = '"');

	//! Returns true if the given string needs to be quoted when written as an identifier
	static bool RequiresQuotes(const string &text, bool allow_caps = true);

	//! Writes a string that is quoted
	static string WriteQuoted(const string &text, char quote = '\'');

	//! Writes a string that is optionally quoted + escaped so it can be used as an identifier
	static string WriteOptionallyQuoted(const string &text, char quote = '"', bool allow_caps = true);
};

} // namespace duckdb



namespace duckdb {

struct QualifiedName {
	string catalog;
	string schema;
	string name;

	//! Parse the (optional) schema and a name from a string in the format of e.g. "schema"."table"; if there is no dot
	//! the schema will be set to INVALID_SCHEMA
	static QualifiedName Parse(const string &input) {
		string catalog;
		string schema;
		string name;
		idx_t idx = 0;
		vector<string> entries;
		string entry;
	normal:
		//! quote
		for (; idx < input.size(); idx++) {
			if (input[idx] == '"') {
				idx++;
				goto quoted;
			} else if (input[idx] == '.') {
				goto separator;
			}
			entry += input[idx];
		}
		goto end;
	separator:
		entries.push_back(entry);
		entry = "";
		idx++;
		goto normal;
	quoted:
		//! look for another quote
		for (; idx < input.size(); idx++) {
			if (input[idx] == '"') {
				//! unquote
				idx++;
				goto normal;
			}
			entry += input[idx];
		}
		throw ParserException("Unterminated quote in qualified name!");
	end:
		if (entries.empty()) {
			catalog = INVALID_CATALOG;
			schema = INVALID_SCHEMA;
			name = entry;
		} else if (entries.size() == 1) {
			catalog = INVALID_CATALOG;
			schema = entries[0];
			name = entry;
		} else if (entries.size() == 2) {
			catalog = entries[0];
			schema = entries[1];
			name = entry;
		} else {
			throw ParserException("Expected catalog.entry, schema.entry or entry: too many entries found");
		}
		return QualifiedName {catalog, schema, name};
	}
};

struct QualifiedColumnName {
	QualifiedColumnName() {
	}
	QualifiedColumnName(string table_p, string column_p) : table(std::move(table_p)), column(std::move(column_p)) {
	}

	string schema;
	string table;
	string column;
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/expression_util.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {
class ParsedExpression;
class Expression;

class ExpressionUtil {
public:
	//! ListEquals: check if a list of two expressions is equal (order is important)
	static bool ListEquals(const vector<unique_ptr<ParsedExpression>> &a,
	                       const vector<unique_ptr<ParsedExpression>> &b);
	static bool ListEquals(const vector<unique_ptr<Expression>> &a, const vector<unique_ptr<Expression>> &b);
	//! SetEquals: check if two sets of expressions are equal (order is not important)
	static bool SetEquals(const vector<unique_ptr<ParsedExpression>> &a, const vector<unique_ptr<ParsedExpression>> &b);
	static bool SetEquals(const vector<unique_ptr<Expression>> &a, const vector<unique_ptr<Expression>> &b);

private:
	template <class T>
	static bool ExpressionListEquals(const vector<unique_ptr<T>> &a, const vector<unique_ptr<T>> &b);
	template <class T, class EXPRESSION_MAP>
	static bool ExpressionSetEquals(const vector<unique_ptr<T>> &a, const vector<unique_ptr<T>> &b);
};

} // namespace duckdb


namespace duckdb {
class Serializer;
class Deserializer;
class FieldWriter;
class FieldReader;
class FormatDeserializer;
class FormatSerializer;

//!  The ParsedExpression class is a base class that can represent any expression
//!  part of a SQL statement.
/*!
 The ParsedExpression class is a base class that can represent any expression
 part of a SQL statement. This is, for example, a column reference in a SELECT
 clause, but also operators, aggregates or filters. The Expression is emitted by the parser and does not contain any
 information about bindings to the catalog or to the types. ParsedExpressions are transformed into regular Expressions
 in the Binder.
 */
class ParsedExpression : public BaseExpression {
public:
	//! Create an Expression
	ParsedExpression(ExpressionType type, ExpressionClass expression_class) : BaseExpression(type, expression_class) {
	}

	//! The location in the query (if any)
	idx_t query_location = DConstants::INVALID_INDEX;

public:
	bool IsAggregate() const override;
	bool IsWindow() const override;
	bool HasSubquery() const override;
	bool IsScalar() const override;
	bool HasParameter() const override;

	bool Equals(const BaseExpression &other) const override;
	hash_t Hash() const override;

	//! Create a copy of this expression
	virtual unique_ptr<ParsedExpression> Copy() const = 0;

	//! Serializes an Expression to a stand-alone binary blob
	void Serialize(Serializer &serializer) const;
	//! Serializes an Expression to a stand-alone binary blob
	virtual void Serialize(FieldWriter &writer) const = 0;

	//! Deserializes a blob back into an Expression [CAN THROW:
	//! SerializationException]
	static unique_ptr<ParsedExpression> Deserialize(Deserializer &source);

	virtual void FormatSerialize(FormatSerializer &serializer) const;
	static unique_ptr<ParsedExpression> FormatDeserialize(FormatDeserializer &deserializer);

	static bool Equals(const unique_ptr<ParsedExpression> &left, const unique_ptr<ParsedExpression> &right);
	static bool ListEquals(const vector<unique_ptr<ParsedExpression>> &left,
	                       const vector<unique_ptr<ParsedExpression>> &right);

protected:
	//! Copy base Expression properties from another expression to this one,
	//! used in Copy method
	void CopyProperties(const ParsedExpression &other) {
		type = other.type;
		expression_class = other.expression_class;
		alias = other.alias;
	}
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/compression_type.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {

enum class CompressionType : uint8_t {
	COMPRESSION_AUTO = 0,
	COMPRESSION_UNCOMPRESSED = 1,
	COMPRESSION_CONSTANT = 2,
	COMPRESSION_RLE = 3,
	COMPRESSION_DICTIONARY = 4,
	COMPRESSION_PFOR_DELTA = 5,
	COMPRESSION_BITPACKING = 6,
	COMPRESSION_FSST = 7,
	COMPRESSION_CHIMP = 8,
	COMPRESSION_PATAS = 9,
	COMPRESSION_COUNT // This has to stay the last entry of the type!
};

vector<string> ListCompressionTypes(void);
CompressionType CompressionTypeFromString(const string &str);
string CompressionTypeToString(CompressionType type);

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/catalog/catalog_entry/table_column_type.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

enum class TableColumnType : uint8_t { STANDARD = 0, GENERATED = 1 };

} // namespace duckdb



namespace duckdb {

struct RenameColumnInfo;
struct RenameTableInfo;

class ColumnDefinition;

//! A column of a table.
class ColumnDefinition {
public:
	DUCKDB_API ColumnDefinition(string name, LogicalType type);
	DUCKDB_API ColumnDefinition(string name, LogicalType type, unique_ptr<ParsedExpression> expression,
	                            TableColumnType category);

	//! The default value of the column (if any)
	unique_ptr<ParsedExpression> default_value;

public:
	//! default_value
	const unique_ptr<ParsedExpression> &DefaultValue() const;
	void SetDefaultValue(unique_ptr<ParsedExpression> default_value);

	//! type
	DUCKDB_API const LogicalType &Type() const;
	LogicalType &TypeMutable();
	void SetType(const LogicalType &type);

	//! name
	DUCKDB_API const string &Name() const;
	void SetName(const string &name);

	//! compression_type
	const duckdb::CompressionType &CompressionType() const;
	void SetCompressionType(duckdb::CompressionType compression_type);

	//! storage_oid
	const storage_t &StorageOid() const;
	void SetStorageOid(storage_t storage_oid);

	LogicalIndex Logical() const;
	PhysicalIndex Physical() const;

	//! oid
	const column_t &Oid() const;
	void SetOid(column_t oid);

	//! category
	const TableColumnType &Category() const;
	//! Whether this column is a Generated Column
	bool Generated() const;
	DUCKDB_API ColumnDefinition Copy() const;

	DUCKDB_API void Serialize(Serializer &serializer) const;
	DUCKDB_API static ColumnDefinition Deserialize(Deserializer &source);

	//===--------------------------------------------------------------------===//
	// Generated Columns (VIRTUAL)
	//===--------------------------------------------------------------------===//

	ParsedExpression &GeneratedExpressionMutable();
	const ParsedExpression &GeneratedExpression() const;
	void SetGeneratedExpression(unique_ptr<ParsedExpression> expression);
	void ChangeGeneratedExpressionType(const LogicalType &type);
	void GetListOfDependencies(vector<string> &dependencies) const;

	string GetName() const;

	LogicalType GetType() const;

private:
	//! The name of the entry
	string name;
	//! The type of the column
	LogicalType type;
	//! Compression Type used for this column
	duckdb::CompressionType compression_type = duckdb::CompressionType::COMPRESSION_AUTO;
	//! The index of the column in the storage of the table
	storage_t storage_oid = DConstants::INVALID_INDEX;
	//! The index of the column in the table
	idx_t oid = DConstants::INVALID_INDEX;
	//! The category of the column
	TableColumnType category = TableColumnType::STANDARD;
	//! Used by Generated Columns
	unique_ptr<ParsedExpression> generated_expression;
};

} // namespace duckdb


namespace duckdb {
class CatalogEntry;
class Catalog;
class ClientContext;
class Expression;
class ExpressionExecutor;
class Transaction;

class AggregateFunction;
class AggregateFunctionSet;
class CopyFunction;
class PragmaFunction;
class PragmaFunctionSet;
class ScalarFunctionSet;
class ScalarFunction;
class TableFunctionSet;
class TableFunction;
class SimpleFunction;

struct PragmaInfo;

//! The default null handling is NULL in, NULL out
enum class FunctionNullHandling : uint8_t { DEFAULT_NULL_HANDLING = 0, SPECIAL_HANDLING = 1 };
enum class FunctionSideEffects : uint8_t { NO_SIDE_EFFECTS = 0, HAS_SIDE_EFFECTS = 1 };

struct FunctionData {
	DUCKDB_API virtual ~FunctionData();

	DUCKDB_API virtual unique_ptr<FunctionData> Copy() const = 0;
	DUCKDB_API virtual bool Equals(const FunctionData &other) const = 0;
	DUCKDB_API static bool Equals(const FunctionData *left, const FunctionData *right);

	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}
	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
	// FIXME: this function should be removed in the future
	template <class TARGET>
	TARGET &CastNoConst() const {
		return const_cast<TARGET &>(reinterpret_cast<const TARGET &>(*this));
	}
};

struct TableFunctionData : public FunctionData {
	// used to pass on projections to table functions that support them. NB, can contain COLUMN_IDENTIFIER_ROW_ID
	vector<idx_t> column_ids;

	DUCKDB_API virtual ~TableFunctionData();

	DUCKDB_API unique_ptr<FunctionData> Copy() const override;
	DUCKDB_API bool Equals(const FunctionData &other) const override;
};

struct PyTableFunctionData : public TableFunctionData {
	//! External dependencies of this table function
	unique_ptr<ExternalDependency> external_dependency;
};

struct FunctionParameters {
	vector<Value> values;
	named_parameter_map_t named_parameters;
};

//! Function is the base class used for any type of function (scalar, aggregate or simple function)
class Function {
public:
	DUCKDB_API explicit Function(string name);
	DUCKDB_API virtual ~Function();

	//! The name of the function
	string name;
	//! Additional Information to specify function from it's name
	string extra_info;

public:
	//! Returns the formatted string name(arg1, arg2, ...)
	DUCKDB_API static string CallToString(const string &name, const vector<LogicalType> &arguments);
	//! Returns the formatted string name(arg1, arg2..) -> return_type
	DUCKDB_API static string CallToString(const string &name, const vector<LogicalType> &arguments,
	                                      const LogicalType &return_type);
	//! Returns the formatted string name(arg1, arg2.., np1=a, np2=b, ...)
	DUCKDB_API static string CallToString(const string &name, const vector<LogicalType> &arguments,
	                                      const named_parameter_type_map_t &named_parameters);

	//! Used in the bind to erase an argument from a function
	DUCKDB_API static void EraseArgument(SimpleFunction &bound_function, vector<unique_ptr<Expression>> &arguments,
	                                     idx_t argument_index);
};

class SimpleFunction : public Function {
public:
	DUCKDB_API SimpleFunction(string name, vector<LogicalType> arguments,
	                          LogicalType varargs = LogicalType(LogicalTypeId::INVALID));
	DUCKDB_API ~SimpleFunction() override;

	//! The set of arguments of the function
	vector<LogicalType> arguments;
	//! The set of original arguments of the function - only set if Function::EraseArgument is called
	//! Used for (de)serialization purposes
	vector<LogicalType> original_arguments;
	//! The type of varargs to support, or LogicalTypeId::INVALID if the function does not accept variable length
	//! arguments
	LogicalType varargs;

public:
	DUCKDB_API virtual string ToString() const;

	DUCKDB_API bool HasVarArgs() const;
};

class SimpleNamedParameterFunction : public SimpleFunction {
public:
	DUCKDB_API SimpleNamedParameterFunction(string name, vector<LogicalType> arguments,
	                                        LogicalType varargs = LogicalType(LogicalTypeId::INVALID));
	DUCKDB_API ~SimpleNamedParameterFunction() override;

	//! The named parameters of the function
	named_parameter_type_map_t named_parameters;

public:
	DUCKDB_API string ToString() const override;
	DUCKDB_API bool HasNamedParameters() const;
};

class BaseScalarFunction : public SimpleFunction {
public:
	DUCKDB_API BaseScalarFunction(string name, vector<LogicalType> arguments, LogicalType return_type,
	                              FunctionSideEffects side_effects,
	                              LogicalType varargs = LogicalType(LogicalTypeId::INVALID),
	                              FunctionNullHandling null_handling = FunctionNullHandling::DEFAULT_NULL_HANDLING);
	DUCKDB_API ~BaseScalarFunction() override;

	//! Return type of the function
	LogicalType return_type;
	//! Whether or not the function has side effects (e.g. sequence increments, random() functions, NOW()). Functions
	//! with side-effects cannot be constant-folded.
	FunctionSideEffects side_effects;
	//! How this function handles NULL values
	FunctionNullHandling null_handling;

public:
	DUCKDB_API hash_t Hash() const;

	DUCKDB_API string ToString() const override;
};

} // namespace duckdb


namespace duckdb {
class Expression;
class ExpressionExecutor;
struct ExpressionExecutorState;
struct FunctionLocalState;

struct ExpressionState {
	ExpressionState(const Expression &expr, ExpressionExecutorState &root);
	virtual ~ExpressionState() {
	}

	const Expression &expr;
	ExpressionExecutorState &root;
	vector<unique_ptr<ExpressionState>> child_states;
	vector<LogicalType> types;
	DataChunk intermediate_chunk;
	CycleCounter profiler;

public:
	void AddChild(Expression *expr);
	void Finalize();
	Allocator &GetAllocator();
	bool HasContext();
	DUCKDB_API ClientContext &GetContext();

	void Verify(ExpressionExecutorState &root);

public:
	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}
	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
};

struct ExecuteFunctionState : public ExpressionState {
	ExecuteFunctionState(const Expression &expr, ExpressionExecutorState &root);
	~ExecuteFunctionState();

	unique_ptr<FunctionLocalState> local_state;

public:
	static optional_ptr<FunctionLocalState> GetFunctionState(ExpressionState &state) {
		return state.Cast<ExecuteFunctionState>().local_state.get();
	}
};

struct ExpressionExecutorState {
	ExpressionExecutorState();

	unique_ptr<ExpressionState> root_state;
	ExpressionExecutor *executor = nullptr;
	CycleCounter profiler;

	void Verify();
};

} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/planner/plan_serialization.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/logical_operator_type.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

//===--------------------------------------------------------------------===//
// Logical Operator Types
//===--------------------------------------------------------------------===//
enum class LogicalOperatorType : uint8_t {
	LOGICAL_INVALID = 0,
	LOGICAL_PROJECTION = 1,
	LOGICAL_FILTER = 2,
	LOGICAL_AGGREGATE_AND_GROUP_BY = 3,
	LOGICAL_WINDOW = 4,
	LOGICAL_UNNEST = 5,
	LOGICAL_LIMIT = 6,
	LOGICAL_ORDER_BY = 7,
	LOGICAL_TOP_N = 8,
	LOGICAL_COPY_TO_FILE = 10,
	LOGICAL_DISTINCT = 11,
	LOGICAL_SAMPLE = 12,
	LOGICAL_LIMIT_PERCENT = 13,
	LOGICAL_PIVOT = 14,

	// -----------------------------
	// Data sources
	// -----------------------------
	LOGICAL_GET = 25,
	LOGICAL_CHUNK_GET = 26,
	LOGICAL_DELIM_GET = 27,
	LOGICAL_EXPRESSION_GET = 28,
	LOGICAL_DUMMY_SCAN = 29,
	LOGICAL_EMPTY_RESULT = 30,
	LOGICAL_CTE_REF = 31,
	// -----------------------------
	// Joins
	// -----------------------------
	LOGICAL_JOIN = 50,
	LOGICAL_DELIM_JOIN = 51,
	LOGICAL_COMPARISON_JOIN = 52,
	LOGICAL_ANY_JOIN = 53,
	LOGICAL_CROSS_PRODUCT = 54,
	LOGICAL_POSITIONAL_JOIN = 55,
	LOGICAL_ASOF_JOIN = 56,
	// -----------------------------
	// SetOps
	// -----------------------------
	LOGICAL_UNION = 75,
	LOGICAL_EXCEPT = 76,
	LOGICAL_INTERSECT = 77,
	LOGICAL_RECURSIVE_CTE = 78,

	// -----------------------------
	// Updates
	// -----------------------------
	LOGICAL_INSERT = 100,
	LOGICAL_DELETE = 101,
	LOGICAL_UPDATE = 102,

	// -----------------------------
	// Schema
	// -----------------------------
	LOGICAL_ALTER = 125,
	LOGICAL_CREATE_TABLE = 126,
	LOGICAL_CREATE_INDEX = 127,
	LOGICAL_CREATE_SEQUENCE = 128,
	LOGICAL_CREATE_VIEW = 129,
	LOGICAL_CREATE_SCHEMA = 130,
	LOGICAL_CREATE_MACRO = 131,
	LOGICAL_DROP = 132,
	LOGICAL_PRAGMA = 133,
	LOGICAL_TRANSACTION = 134,
	LOGICAL_CREATE_TYPE = 135,
	LOGICAL_ATTACH = 136,
	LOGICAL_DETACH = 137,

	// -----------------------------
	// Explain
	// -----------------------------
	LOGICAL_EXPLAIN = 150,

	// -----------------------------
	// Show
	// -----------------------------
	LOGICAL_SHOW = 160,

	// -----------------------------
	// Helpers
	// -----------------------------
	LOGICAL_PREPARE = 175,
	LOGICAL_EXECUTE = 176,
	LOGICAL_EXPORT = 177,
	LOGICAL_VACUUM = 178,
	LOGICAL_SET = 179,
	LOGICAL_LOAD = 180,
	LOGICAL_RESET = 181,

	LOGICAL_EXTENSION_OPERATOR = 255
};

DUCKDB_API string LogicalOperatorToString(LogicalOperatorType type);

} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/planner/bound_parameter_map.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {

struct BoundParameterData;

using bound_parameter_map_t = unordered_map<idx_t, shared_ptr<BoundParameterData>>;

} // namespace duckdb


namespace duckdb {
class ClientContext;
class LogicalOperator;
struct BoundParameterData;

struct PlanDeserializationState {
	PlanDeserializationState(ClientContext &context);
	~PlanDeserializationState();

	ClientContext &context;
	bound_parameter_map_t parameter_data;
};

struct LogicalDeserializationState {
	LogicalDeserializationState(PlanDeserializationState &gstate, LogicalOperatorType type,
	                            vector<unique_ptr<LogicalOperator>> &children);

	PlanDeserializationState &gstate;
	LogicalOperatorType type;
	vector<unique_ptr<LogicalOperator>> &children;
};

struct ExpressionDeserializationState {
	ExpressionDeserializationState(PlanDeserializationState &gstate, ExpressionType type);

	PlanDeserializationState &gstate;
	ExpressionType type;
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/storage/statistics/base_statistics.hpp
//
//
//===----------------------------------------------------------------------===//





//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/operator/comparison_operators.hpp
//
//
//===----------------------------------------------------------------------===//





//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/types/hugeint.hpp
//
//
//===----------------------------------------------------------------------===//












namespace duckdb {

//! Returns the PhysicalType for the given type
template <class T>
PhysicalType GetTypeId() {
	if (std::is_same<T, bool>()) {
		return PhysicalType::BOOL;
	} else if (std::is_same<T, int8_t>()) {
		return PhysicalType::INT8;
	} else if (std::is_same<T, int16_t>()) {
		return PhysicalType::INT16;
	} else if (std::is_same<T, int32_t>()) {
		return PhysicalType::INT32;
	} else if (std::is_same<T, int64_t>()) {
		return PhysicalType::INT64;
	} else if (std::is_same<T, uint8_t>()) {
		return PhysicalType::UINT8;
	} else if (std::is_same<T, uint16_t>()) {
		return PhysicalType::UINT16;
	} else if (std::is_same<T, uint32_t>()) {
		return PhysicalType::UINT32;
	} else if (std::is_same<T, uint64_t>()) {
		return PhysicalType::UINT64;
	} else if (std::is_same<T, hugeint_t>()) {
		return PhysicalType::INT128;
	} else if (std::is_same<T, date_t>()) {
		return PhysicalType::INT32;
	} else if (std::is_same<T, dtime_t>()) {
		return PhysicalType::INT64;
	} else if (std::is_same<T, timestamp_t>()) {
		return PhysicalType::INT64;
	} else if (std::is_same<T, float>()) {
		return PhysicalType::FLOAT;
	} else if (std::is_same<T, double>()) {
		return PhysicalType::DOUBLE;
	} else if (std::is_same<T, const char *>() || std::is_same<T, char *>() || std::is_same<T, string_t>()) {
		return PhysicalType::VARCHAR;
	} else if (std::is_same<T, interval_t>()) {
		return PhysicalType::INTERVAL;
	} else {
		return PhysicalType::INVALID;
	}
}

template <class T>
bool TypeIsNumber() {
	return std::is_integral<T>() || std::is_floating_point<T>() || std::is_same<T, hugeint_t>();
}

template <class T>
bool IsValidType() {
	return GetTypeId<T>() != PhysicalType::INVALID;
}

template <class T>
bool IsIntegerType() {
	return TypeIsIntegral(GetTypeId<T>());
}

} // namespace duckdb




namespace duckdb {

//! The Hugeint class contains static operations for the INT128 type
class Hugeint {
public:
	//! Convert a hugeint object to a string
	static string ToString(hugeint_t input);

	template <class T>
	DUCKDB_API static bool TryCast(hugeint_t input, T &result);

	template <class T>
	static T Cast(hugeint_t input) {
		T result = 0;
		TryCast(input, result);
		return result;
	}

	template <class T>
	static bool TryConvert(T value, hugeint_t &result);

	template <class T>
	static hugeint_t Convert(T value) {
		hugeint_t result;
		if (!TryConvert(value, result)) { // LCOV_EXCL_START
			throw ValueOutOfRangeException(double(value), GetTypeId<T>(), GetTypeId<hugeint_t>());
		} // LCOV_EXCL_STOP
		return result;
	}

	static void NegateInPlace(hugeint_t &input) {
		if (input.upper == NumericLimits<int64_t>::Minimum() && input.lower == 0) {
			throw OutOfRangeException("HUGEINT is out of range");
		}
		input.lower = NumericLimits<uint64_t>::Maximum() - input.lower + 1;
		input.upper = -1 - input.upper + (input.lower == 0);
	}
	static hugeint_t Negate(hugeint_t input) {
		NegateInPlace(input);
		return input;
	}

	static bool TryMultiply(hugeint_t lhs, hugeint_t rhs, hugeint_t &result);

	static hugeint_t Add(hugeint_t lhs, hugeint_t rhs);
	static hugeint_t Subtract(hugeint_t lhs, hugeint_t rhs);
	static hugeint_t Multiply(hugeint_t lhs, hugeint_t rhs);
	static hugeint_t Divide(hugeint_t lhs, hugeint_t rhs);
	static hugeint_t Modulo(hugeint_t lhs, hugeint_t rhs);

	// DivMod -> returns the result of the division (lhs / rhs), and fills up the remainder
	static hugeint_t DivMod(hugeint_t lhs, hugeint_t rhs, hugeint_t &remainder);
	// DivMod but lhs MUST be positive, and rhs is a uint64_t
	static hugeint_t DivModPositive(hugeint_t lhs, uint64_t rhs, uint64_t &remainder);

	static bool AddInPlace(hugeint_t &lhs, hugeint_t rhs);
	static bool SubtractInPlace(hugeint_t &lhs, hugeint_t rhs);

	// comparison operators
	// note that everywhere here we intentionally use bitwise ops
	// this is because they seem to be consistently much faster (benchmarked on a Macbook Pro)
	static bool Equals(hugeint_t lhs, hugeint_t rhs) {
		int lower_equals = lhs.lower == rhs.lower;
		int upper_equals = lhs.upper == rhs.upper;
		return lower_equals & upper_equals;
	}
	static bool NotEquals(hugeint_t lhs, hugeint_t rhs) {
		int lower_not_equals = lhs.lower != rhs.lower;
		int upper_not_equals = lhs.upper != rhs.upper;
		return lower_not_equals | upper_not_equals;
	}
	static bool GreaterThan(hugeint_t lhs, hugeint_t rhs) {
		int upper_bigger = lhs.upper > rhs.upper;
		int upper_equal = lhs.upper == rhs.upper;
		int lower_bigger = lhs.lower > rhs.lower;
		return upper_bigger | (upper_equal & lower_bigger);
	}
	static bool GreaterThanEquals(hugeint_t lhs, hugeint_t rhs) {
		int upper_bigger = lhs.upper > rhs.upper;
		int upper_equal = lhs.upper == rhs.upper;
		int lower_bigger_equals = lhs.lower >= rhs.lower;
		return upper_bigger | (upper_equal & lower_bigger_equals);
	}
	static bool LessThan(hugeint_t lhs, hugeint_t rhs) {
		int upper_smaller = lhs.upper < rhs.upper;
		int upper_equal = lhs.upper == rhs.upper;
		int lower_smaller = lhs.lower < rhs.lower;
		return upper_smaller | (upper_equal & lower_smaller);
	}
	static bool LessThanEquals(hugeint_t lhs, hugeint_t rhs) {
		int upper_smaller = lhs.upper < rhs.upper;
		int upper_equal = lhs.upper == rhs.upper;
		int lower_smaller_equals = lhs.lower <= rhs.lower;
		return upper_smaller | (upper_equal & lower_smaller_equals);
	}
	static const hugeint_t POWERS_OF_TEN[40];
};

template <>
DUCKDB_API bool Hugeint::TryCast(hugeint_t input, int8_t &result);
template <>
DUCKDB_API bool Hugeint::TryCast(hugeint_t input, int16_t &result);
template <>
DUCKDB_API bool Hugeint::TryCast(hugeint_t input, int32_t &result);
template <>
DUCKDB_API bool Hugeint::TryCast(hugeint_t input, int64_t &result);
template <>
DUCKDB_API bool Hugeint::TryCast(hugeint_t input, uint8_t &result);
template <>
DUCKDB_API bool Hugeint::TryCast(hugeint_t input, uint16_t &result);
template <>
DUCKDB_API bool Hugeint::TryCast(hugeint_t input, uint32_t &result);
template <>
DUCKDB_API bool Hugeint::TryCast(hugeint_t input, uint64_t &result);
template <>
DUCKDB_API bool Hugeint::TryCast(hugeint_t input, hugeint_t &result);
template <>
DUCKDB_API bool Hugeint::TryCast(hugeint_t input, float &result);
template <>
DUCKDB_API bool Hugeint::TryCast(hugeint_t input, double &result);
template <>
DUCKDB_API bool Hugeint::TryCast(hugeint_t input, long double &result);

template <>
bool Hugeint::TryConvert(int8_t value, hugeint_t &result);
template <>
bool Hugeint::TryConvert(int16_t value, hugeint_t &result);
template <>
bool Hugeint::TryConvert(int32_t value, hugeint_t &result);
template <>
bool Hugeint::TryConvert(int64_t value, hugeint_t &result);
template <>
bool Hugeint::TryConvert(uint8_t value, hugeint_t &result);
template <>
bool Hugeint::TryConvert(uint16_t value, hugeint_t &result);
template <>
bool Hugeint::TryConvert(uint32_t value, hugeint_t &result);
template <>
bool Hugeint::TryConvert(uint64_t value, hugeint_t &result);
template <>
bool Hugeint::TryConvert(float value, hugeint_t &result);
template <>
bool Hugeint::TryConvert(double value, hugeint_t &result);
template <>
bool Hugeint::TryConvert(long double value, hugeint_t &result);
template <>
bool Hugeint::TryConvert(const char *value, hugeint_t &result);

} // namespace duckdb




#include <cstring>

namespace duckdb {

//===--------------------------------------------------------------------===//
// Comparison Operations
//===--------------------------------------------------------------------===//
struct Equals {
	template <class T>
	static inline bool Operation(const T &left, const T &right) {
		return left == right;
	}
};
struct NotEquals {
	template <class T>
	static inline bool Operation(const T &left, const T &right) {
		return !Equals::Operation(left, right);
	}
};

struct GreaterThan {
	template <class T>
	static inline bool Operation(const T &left, const T &right) {
		return left > right;
	}
};

struct GreaterThanEquals {
	template <class T>
	static inline bool Operation(const T &left, const T &right) {
		return !GreaterThan::Operation(right, left);
	}
};

struct LessThan {
	template <class T>
	static inline bool Operation(const T &left, const T &right) {
		return GreaterThan::Operation(right, left);
	}
};

struct LessThanEquals {
	template <class T>
	static inline bool Operation(const T &left, const T &right) {
		return !GreaterThan::Operation(left, right);
	}
};

template <>
DUCKDB_API bool Equals::Operation(const float &left, const float &right);
template <>
DUCKDB_API bool Equals::Operation(const double &left, const double &right);

template <>
DUCKDB_API bool GreaterThan::Operation(const float &left, const float &right);
template <>
DUCKDB_API bool GreaterThan::Operation(const double &left, const double &right);

template <>
DUCKDB_API bool GreaterThanEquals::Operation(const float &left, const float &right);
template <>
DUCKDB_API bool GreaterThanEquals::Operation(const double &left, const double &right);

// Distinct semantics are from Postgres record sorting. NULL = NULL and not-NULL < NULL
// Deferring to the non-distinct operations removes the need for further specialisation.
// TODO: To reverse the semantics, swap left_null and right_null for comparisons
struct DistinctFrom {
	template <class T>
	static inline bool Operation(const T &left, const T &right, bool left_null, bool right_null) {
		if (left_null || right_null) {
			return left_null != right_null;
		}
		return NotEquals::Operation(left, right);
	}
};

struct NotDistinctFrom {
	template <class T>
	static inline bool Operation(const T &left, const T &right, bool left_null, bool right_null) {
		return !DistinctFrom::Operation(left, right, left_null, right_null);
	}
};

struct DistinctGreaterThan {
	template <class T>
	static inline bool Operation(const T &left, const T &right, bool left_null, bool right_null) {
		if (left_null || right_null) {
			return !right_null;
		}
		return GreaterThan::Operation(left, right);
	}
};

struct DistinctGreaterThanNullsFirst {
	template <class T>
	static inline bool Operation(const T &left, const T &right, bool left_null, bool right_null) {
		return DistinctGreaterThan::Operation(left, right, right_null, left_null);
	}
};

struct DistinctGreaterThanEquals {
	template <class T>
	static inline bool Operation(const T &left, const T &right, bool left_null, bool right_null) {
		return !DistinctGreaterThan::Operation(right, left, right_null, left_null);
	}
};

struct DistinctLessThan {
	template <class T>
	static inline bool Operation(const T &left, const T &right, bool left_null, bool right_null) {
		return DistinctGreaterThan::Operation(right, left, right_null, left_null);
	}
};

struct DistinctLessThanNullsFirst {
	template <class T>
	static inline bool Operation(const T &left, const T &right, bool left_null, bool right_null) {
		return DistinctGreaterThan::Operation(right, left, left_null, right_null);
	}
};

struct DistinctLessThanEquals {
	template <class T>
	static inline bool Operation(const T &left, const T &right, bool left_null, bool right_null) {
		return !DistinctGreaterThan::Operation(left, right, left_null, right_null);
	}
};

//===--------------------------------------------------------------------===//
// Specialized Boolean Comparison Operators
//===--------------------------------------------------------------------===//
template <>
inline bool GreaterThan::Operation(const bool &left, const bool &right) {
	return !right && left;
}
//===--------------------------------------------------------------------===//
// Specialized String Comparison Operations
//===--------------------------------------------------------------------===//
template <>
inline bool Equals::Operation(const string_t &left, const string_t &right) {
	return left == right;
}

template <>
inline bool GreaterThan::Operation(const string_t &left, const string_t &right) {
	return left > right;
}

//===--------------------------------------------------------------------===//
// Specialized Interval Comparison Operators
//===--------------------------------------------------------------------===//
template <>
inline bool Equals::Operation(const interval_t &left, const interval_t &right) {
	return Interval::Equals(left, right);
}
template <>
inline bool GreaterThan::Operation(const interval_t &left, const interval_t &right) {
	return Interval::GreaterThan(left, right);
}

inline bool operator<(const interval_t &lhs, const interval_t &rhs) {
	return LessThan::Operation(lhs, rhs);
}

//===--------------------------------------------------------------------===//
// Specialized Hugeint Comparison Operators
//===--------------------------------------------------------------------===//
template <>
inline bool Equals::Operation(const hugeint_t &left, const hugeint_t &right) {
	return Hugeint::Equals(left, right);
}
template <>
inline bool GreaterThan::Operation(const hugeint_t &left, const hugeint_t &right) {
	return Hugeint::GreaterThan(left, right);
}
} // namespace duckdb



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/storage/statistics/numeric_stats.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/storage/statistics/numeric_stats_union.hpp
//
//
//===----------------------------------------------------------------------===//







namespace duckdb {

struct NumericValueUnion {
	union Val {
		bool boolean;
		int8_t tinyint;
		int16_t smallint;
		int32_t integer;
		int64_t bigint;
		uint8_t utinyint;
		uint16_t usmallint;
		uint32_t uinteger;
		uint64_t ubigint;
		hugeint_t hugeint;
		float float_;
		double double_;
	} value_;

	template <class T>
	T &GetReferenceUnsafe();
};

template <>
DUCKDB_API bool &NumericValueUnion::GetReferenceUnsafe();
template <>
DUCKDB_API int8_t &NumericValueUnion::GetReferenceUnsafe();
template <>
DUCKDB_API int16_t &NumericValueUnion::GetReferenceUnsafe();
template <>
DUCKDB_API int32_t &NumericValueUnion::GetReferenceUnsafe();
template <>
DUCKDB_API int64_t &NumericValueUnion::GetReferenceUnsafe();
template <>
DUCKDB_API hugeint_t &NumericValueUnion::GetReferenceUnsafe();
template <>
DUCKDB_API uint8_t &NumericValueUnion::GetReferenceUnsafe();
template <>
DUCKDB_API uint16_t &NumericValueUnion::GetReferenceUnsafe();
template <>
DUCKDB_API uint32_t &NumericValueUnion::GetReferenceUnsafe();
template <>
DUCKDB_API uint64_t &NumericValueUnion::GetReferenceUnsafe();
template <>
DUCKDB_API float &NumericValueUnion::GetReferenceUnsafe();
template <>
DUCKDB_API double &NumericValueUnion::GetReferenceUnsafe();

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/filter_propagate_result.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

enum class FilterPropagateResult : uint8_t {
	NO_PRUNING_POSSIBLE = 0,
	FILTER_ALWAYS_TRUE = 1,
	FILTER_ALWAYS_FALSE = 2,
	FILTER_TRUE_OR_NULL = 3,
	FILTER_FALSE_OR_NULL = 4
};

} // namespace duckdb





namespace duckdb {
class BaseStatistics;
class FieldWriter;
class FieldReader;
struct SelectionVector;
class Vector;

struct NumericStatsData {
	//! Whether or not the value has a max value
	bool has_min;
	//! Whether or not the segment has a min value
	bool has_max;
	//! The minimum value of the segment
	NumericValueUnion min;
	//! The maximum value of the segment
	NumericValueUnion max;
};

struct NumericStats {
	//! Unknown statistics - i.e. "has_min" is false, "has_max" is false
	DUCKDB_API static BaseStatistics CreateUnknown(LogicalType type);
	//! Empty statistics - i.e. "min = MaxValue<type>, max = MinValue<type>"
	DUCKDB_API static BaseStatistics CreateEmpty(LogicalType type);

	//! Returns true if the stats has a constant value
	DUCKDB_API static bool IsConstant(const BaseStatistics &stats);
	//! Returns true if the stats has both a min and max value defined
	DUCKDB_API static bool HasMinMax(const BaseStatistics &stats);
	//! Returns true if the stats has a min value defined
	DUCKDB_API static bool HasMin(const BaseStatistics &stats);
	//! Returns true if the stats has a max value defined
	DUCKDB_API static bool HasMax(const BaseStatistics &stats);
	//! Returns the min value - throws an exception if there is no min value
	DUCKDB_API static Value Min(const BaseStatistics &stats);
	//! Returns the max value - throws an exception if there is no max value
	DUCKDB_API static Value Max(const BaseStatistics &stats);
	//! Sets the min value of the statistics
	DUCKDB_API static void SetMin(BaseStatistics &stats, const Value &val);
	//! Sets the max value of the statistics
	DUCKDB_API static void SetMax(BaseStatistics &stats, const Value &val);

	//! Check whether or not a given comparison with a constant could possibly be satisfied by rows given the statistics
	DUCKDB_API static FilterPropagateResult CheckZonemap(const BaseStatistics &stats, ExpressionType comparison_type,
	                                                     const Value &constant);

	DUCKDB_API static void Merge(BaseStatistics &stats, const BaseStatistics &other_p);

	DUCKDB_API static void Serialize(const BaseStatistics &stats, FieldWriter &writer);
	DUCKDB_API static BaseStatistics Deserialize(FieldReader &reader, LogicalType type);

	DUCKDB_API static string ToString(const BaseStatistics &stats);

	template <class T>
	static inline void UpdateValue(T new_value, T &min, T &max) {
		if (LessThan::Operation(new_value, min)) {
			min = new_value;
		}
		if (GreaterThan::Operation(new_value, max)) {
			max = new_value;
		}
	}

	template <class T>
	static inline void Update(BaseStatistics &stats, T new_value) {
		auto &nstats = NumericStats::GetDataUnsafe(stats);
		UpdateValue<T>(new_value, nstats.min.GetReferenceUnsafe<T>(), nstats.max.GetReferenceUnsafe<T>());
	}

	static void Verify(const BaseStatistics &stats, Vector &vector, const SelectionVector &sel, idx_t count);

	template <class T>
	static T GetMin(const BaseStatistics &stats) {
		return NumericStats::Min(stats).GetValueUnsafe<T>();
	}
	template <class T>
	static T GetMax(const BaseStatistics &stats) {
		return NumericStats::Max(stats).GetValueUnsafe<T>();
	}
	template <class T>
	static T GetMinUnsafe(const BaseStatistics &stats);
	template <class T>
	static T GetMaxUnsafe(const BaseStatistics &stats);

private:
	static NumericStatsData &GetDataUnsafe(BaseStatistics &stats);
	static const NumericStatsData &GetDataUnsafe(const BaseStatistics &stats);
	static Value MinOrNull(const BaseStatistics &stats);
	static Value MaxOrNull(const BaseStatistics &stats);
	template <class T>
	static void TemplatedVerify(const BaseStatistics &stats, Vector &vector, const SelectionVector &sel, idx_t count);
};

template <>
void NumericStats::Update<interval_t>(BaseStatistics &stats, interval_t new_value);
template <>
void NumericStats::Update<list_entry_t>(BaseStatistics &stats, list_entry_t new_value);

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/storage/statistics/string_stats.hpp
//
//
//===----------------------------------------------------------------------===//










namespace duckdb {
class BaseStatistics;
class FieldWriter;
class FieldReader;
struct SelectionVector;
class Vector;

struct StringStatsData {
	constexpr static uint32_t MAX_STRING_MINMAX_SIZE = 8;

	//! The minimum value of the segment, potentially truncated
	data_t min[MAX_STRING_MINMAX_SIZE];
	//! The maximum value of the segment, potentially truncated
	data_t max[MAX_STRING_MINMAX_SIZE];
	//! Whether or not the column can contain unicode characters
	bool has_unicode;
	//! Whether or not the maximum string length is known
	bool has_max_string_length;
	//! The maximum string length in bytes
	uint32_t max_string_length;
};

struct StringStats {
	//! Unknown statistics - i.e. "has_unicode" is true, "max_string_length" is unknown, "min" is \0, max is \xFF
	DUCKDB_API static BaseStatistics CreateUnknown(LogicalType type);
	//! Empty statistics - i.e. "has_unicode" is false, "max_string_length" is 0, "min" is \xFF, max is \x00
	DUCKDB_API static BaseStatistics CreateEmpty(LogicalType type);
	//! Whether or not the statistics have a maximum string length defined
	DUCKDB_API static bool HasMaxStringLength(const BaseStatistics &stats);
	//! Returns the maximum string length, or throws an exception if !HasMaxStringLength()
	DUCKDB_API static uint32_t MaxStringLength(const BaseStatistics &stats);
	//! Whether or not the strings can contain unicode
	DUCKDB_API static bool CanContainUnicode(const BaseStatistics &stats);

	//! Resets the max string length so HasMaxStringLength() is false
	DUCKDB_API static void ResetMaxStringLength(BaseStatistics &stats);
	//! FIXME: make this part of Set on statistics
	DUCKDB_API static void SetContainsUnicode(BaseStatistics &stats);

	DUCKDB_API static void Serialize(const BaseStatistics &stats, FieldWriter &writer);
	DUCKDB_API static BaseStatistics Deserialize(FieldReader &reader, LogicalType type);

	DUCKDB_API static string ToString(const BaseStatistics &stats);

	DUCKDB_API static FilterPropagateResult CheckZonemap(const BaseStatistics &stats, ExpressionType comparison_type,
	                                                     const string &value);

	DUCKDB_API static void Update(BaseStatistics &stats, const string_t &value);
	DUCKDB_API static void Merge(BaseStatistics &stats, const BaseStatistics &other);
	DUCKDB_API static void Verify(const BaseStatistics &stats, Vector &vector, const SelectionVector &sel, idx_t count);

private:
	static StringStatsData &GetDataUnsafe(BaseStatistics &stats);
	static const StringStatsData &GetDataUnsafe(const BaseStatistics &stats);
};

} // namespace duckdb


namespace duckdb {
struct SelectionVector;

class Serializer;
class Deserializer;
class FieldWriter;
class FieldReader;
class Vector;
struct UnifiedVectorFormat;

enum class StatsInfo : uint8_t {
	CAN_HAVE_NULL_VALUES = 0,
	CANNOT_HAVE_NULL_VALUES = 1,
	CAN_HAVE_VALID_VALUES = 2,
	CANNOT_HAVE_VALID_VALUES = 3,
	CAN_HAVE_NULL_AND_VALID_VALUES = 4
};

enum class StatisticsType : uint8_t { NUMERIC_STATS, STRING_STATS, LIST_STATS, STRUCT_STATS, BASE_STATS };

class BaseStatistics {
	friend struct NumericStats;
	friend struct StringStats;
	friend struct StructStats;
	friend struct ListStats;

public:
	DUCKDB_API ~BaseStatistics();
	// disable copy constructors
	BaseStatistics(const BaseStatistics &other) = delete;
	BaseStatistics &operator=(const BaseStatistics &) = delete;
	//! enable move constructors
	DUCKDB_API BaseStatistics(BaseStatistics &&other) noexcept;
	DUCKDB_API BaseStatistics &operator=(BaseStatistics &&) noexcept;

public:
	//! Creates a set of statistics for data that is unknown, i.e. "has_null" is true, "has_no_null" is true, etc
	//! This can be used in case nothing is known about the data - or can be used as a baseline when only a few things
	//! are known
	static BaseStatistics CreateUnknown(LogicalType type);
	//! Creates statistics for an empty database, i.e. "has_null" is false, "has_no_null" is false, etc
	//! This is used when incrementally constructing statistics by constantly adding new values
	static BaseStatistics CreateEmpty(LogicalType type);

	DUCKDB_API StatisticsType GetStatsType() const;
	DUCKDB_API static StatisticsType GetStatsType(const LogicalType &type);

	DUCKDB_API bool CanHaveNull() const;
	DUCKDB_API bool CanHaveNoNull() const;

	void SetDistinctCount(idx_t distinct_count);

	bool IsConstant() const;

	const LogicalType &GetType() const {
		return type;
	}

	void Set(StatsInfo info);
	void CombineValidity(BaseStatistics &left, BaseStatistics &right);
	void CopyValidity(BaseStatistics &stats);
	inline void SetHasNull() {
		has_null = true;
	}
	inline void SetHasNoNull() {
		has_no_null = true;
	}

	void Merge(const BaseStatistics &other);

	void Copy(const BaseStatistics &other);

	BaseStatistics Copy() const;
	unique_ptr<BaseStatistics> ToUnique() const;
	void CopyBase(const BaseStatistics &orig);

	void Serialize(Serializer &serializer) const;
	void Serialize(FieldWriter &writer) const;

	idx_t GetDistinctCount();

	static BaseStatistics Deserialize(Deserializer &source, LogicalType type);

	//! Verify that a vector does not violate the statistics
	void Verify(Vector &vector, const SelectionVector &sel, idx_t count) const;
	void Verify(Vector &vector, idx_t count) const;

	string ToString() const;

	static BaseStatistics FromConstant(const Value &input);

private:
	BaseStatistics();
	explicit BaseStatistics(LogicalType type);

	static void Construct(BaseStatistics &stats, LogicalType type);

	void InitializeUnknown();
	void InitializeEmpty();

	static BaseStatistics CreateUnknownType(LogicalType type);
	static BaseStatistics CreateEmptyType(LogicalType type);
	static BaseStatistics DeserializeType(FieldReader &reader, LogicalType type);
	static BaseStatistics FromConstantType(const Value &input);

private:
	//! The type of the logical segment
	LogicalType type;
	//! Whether or not the segment can contain NULL values
	bool has_null;
	//! Whether or not the segment can contain values that are not null
	bool has_no_null;
	// estimate that one may have even if distinct_stats==nullptr
	idx_t distinct_count;
	//! Numeric and String stats
	union {
		//! Numeric stats data, for numeric stats
		NumericStatsData numeric_data;
		//! String stats data, for string stats
		StringStatsData string_data;
	} stats_union;
	//! Child stats (for LIST and STRUCT)
	unsafe_unique_array<BaseStatistics> child_stats;
};

} // namespace duckdb



namespace duckdb {

struct FunctionLocalState {
	DUCKDB_API virtual ~FunctionLocalState();

	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}
	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
};

class Binder;
class BoundFunctionExpression;
class DependencyList;
class ScalarFunctionCatalogEntry;

struct FunctionStatisticsInput {
	FunctionStatisticsInput(BoundFunctionExpression &expr_p, optional_ptr<FunctionData> bind_data_p,
	                        vector<BaseStatistics> &child_stats_p, unique_ptr<Expression> *expr_ptr_p)
	    : expr(expr_p), bind_data(bind_data_p), child_stats(child_stats_p), expr_ptr(expr_ptr_p) {
	}

	BoundFunctionExpression &expr;
	optional_ptr<FunctionData> bind_data;
	vector<BaseStatistics> &child_stats;
	unique_ptr<Expression> *expr_ptr;
};

//! The type used for scalar functions
typedef std::function<void(DataChunk &, ExpressionState &, Vector &)> scalar_function_t;
//! Binds the scalar function and creates the function data
typedef unique_ptr<FunctionData> (*bind_scalar_function_t)(ClientContext &context, ScalarFunction &bound_function,
                                                           vector<unique_ptr<Expression>> &arguments);
typedef unique_ptr<FunctionLocalState> (*init_local_state_t)(ExpressionState &state,
                                                             const BoundFunctionExpression &expr,
                                                             FunctionData *bind_data);
typedef unique_ptr<BaseStatistics> (*function_statistics_t)(ClientContext &context, FunctionStatisticsInput &input);
//! Adds the dependencies of this BoundFunctionExpression to the set of dependencies
typedef void (*dependency_function_t)(BoundFunctionExpression &expr, DependencyList &dependencies);

typedef void (*function_serialize_t)(FieldWriter &writer, const FunctionData *bind_data,
                                     const ScalarFunction &function);
typedef unique_ptr<FunctionData> (*function_deserialize_t)(PlanDeserializationState &state, FieldReader &reader,
                                                           ScalarFunction &function);

class ScalarFunction : public BaseScalarFunction {
public:
	DUCKDB_API ScalarFunction(string name, vector<LogicalType> arguments, LogicalType return_type,
	                          scalar_function_t function, bind_scalar_function_t bind = nullptr,
	                          dependency_function_t dependency = nullptr, function_statistics_t statistics = nullptr,
	                          init_local_state_t init_local_state = nullptr,
	                          LogicalType varargs = LogicalType(LogicalTypeId::INVALID),
	                          FunctionSideEffects side_effects = FunctionSideEffects::NO_SIDE_EFFECTS,
	                          FunctionNullHandling null_handling = FunctionNullHandling::DEFAULT_NULL_HANDLING);

	DUCKDB_API ScalarFunction(vector<LogicalType> arguments, LogicalType return_type, scalar_function_t function,
	                          bind_scalar_function_t bind = nullptr, dependency_function_t dependency = nullptr,
	                          function_statistics_t statistics = nullptr, init_local_state_t init_local_state = nullptr,
	                          LogicalType varargs = LogicalType(LogicalTypeId::INVALID),
	                          FunctionSideEffects side_effects = FunctionSideEffects::NO_SIDE_EFFECTS,
	                          FunctionNullHandling null_handling = FunctionNullHandling::DEFAULT_NULL_HANDLING);

	//! The main scalar function to execute
	scalar_function_t function;
	//! The bind function (if any)
	bind_scalar_function_t bind;
	//! Init thread local state for the function (if any)
	init_local_state_t init_local_state;
	//! The dependency function (if any)
	dependency_function_t dependency;
	//! The statistics propagation function (if any)
	function_statistics_t statistics;

	function_serialize_t serialize;
	function_deserialize_t deserialize;

	DUCKDB_API bool operator==(const ScalarFunction &rhs) const;
	DUCKDB_API bool operator!=(const ScalarFunction &rhs) const;

	DUCKDB_API bool Equal(const ScalarFunction &rhs) const;

private:
	bool CompareScalarFunctionT(const scalar_function_t &other) const;

public:
	DUCKDB_API static void NopFunction(DataChunk &input, ExpressionState &state, Vector &result);

	template <class TA, class TR, class OP>
	static void UnaryFunction(DataChunk &input, ExpressionState &state, Vector &result) {
		D_ASSERT(input.ColumnCount() >= 1);
		UnaryExecutor::Execute<TA, TR, OP>(input.data[0], result, input.size());
	}

	template <class TA, class TB, class TR, class OP>
	static void BinaryFunction(DataChunk &input, ExpressionState &state, Vector &result) {
		D_ASSERT(input.ColumnCount() == 2);
		BinaryExecutor::ExecuteStandard<TA, TB, TR, OP>(input.data[0], input.data[1], result, input.size());
	}

	template <class TA, class TB, class TC, class TR, class OP>
	static void TernaryFunction(DataChunk &input, ExpressionState &state, Vector &result) {
		D_ASSERT(input.ColumnCount() == 3);
		TernaryExecutor::ExecuteStandard<TA, TB, TC, TR, OP>(input.data[0], input.data[1], input.data[2], result,
		                                                     input.size());
	}

public:
	template <class OP>
	static scalar_function_t GetScalarUnaryFunction(LogicalType type) {
		scalar_function_t function;
		switch (type.id()) {
		case LogicalTypeId::TINYINT:
			function = &ScalarFunction::UnaryFunction<int8_t, int8_t, OP>;
			break;
		case LogicalTypeId::SMALLINT:
			function = &ScalarFunction::UnaryFunction<int16_t, int16_t, OP>;
			break;
		case LogicalTypeId::INTEGER:
			function = &ScalarFunction::UnaryFunction<int32_t, int32_t, OP>;
			break;
		case LogicalTypeId::BIGINT:
			function = &ScalarFunction::UnaryFunction<int64_t, int64_t, OP>;
			break;
		case LogicalTypeId::UTINYINT:
			function = &ScalarFunction::UnaryFunction<uint8_t, uint8_t, OP>;
			break;
		case LogicalTypeId::USMALLINT:
			function = &ScalarFunction::UnaryFunction<uint16_t, uint16_t, OP>;
			break;
		case LogicalTypeId::UINTEGER:
			function = &ScalarFunction::UnaryFunction<uint32_t, uint32_t, OP>;
			break;
		case LogicalTypeId::UBIGINT:
			function = &ScalarFunction::UnaryFunction<uint64_t, uint64_t, OP>;
			break;
		case LogicalTypeId::HUGEINT:
			function = &ScalarFunction::UnaryFunction<hugeint_t, hugeint_t, OP>;
			break;
		case LogicalTypeId::FLOAT:
			function = &ScalarFunction::UnaryFunction<float, float, OP>;
			break;
		case LogicalTypeId::DOUBLE:
			function = &ScalarFunction::UnaryFunction<double, double, OP>;
			break;
		default:
			throw InternalException("Unimplemented type for GetScalarUnaryFunction");
		}
		return function;
	}

	template <class TR, class OP>
	static scalar_function_t GetScalarUnaryFunctionFixedReturn(LogicalType type) {
		scalar_function_t function;
		switch (type.id()) {
		case LogicalTypeId::TINYINT:
			function = &ScalarFunction::UnaryFunction<int8_t, TR, OP>;
			break;
		case LogicalTypeId::SMALLINT:
			function = &ScalarFunction::UnaryFunction<int16_t, TR, OP>;
			break;
		case LogicalTypeId::INTEGER:
			function = &ScalarFunction::UnaryFunction<int32_t, TR, OP>;
			break;
		case LogicalTypeId::BIGINT:
			function = &ScalarFunction::UnaryFunction<int64_t, TR, OP>;
			break;
		case LogicalTypeId::UTINYINT:
			function = &ScalarFunction::UnaryFunction<uint8_t, TR, OP>;
			break;
		case LogicalTypeId::USMALLINT:
			function = &ScalarFunction::UnaryFunction<uint16_t, TR, OP>;
			break;
		case LogicalTypeId::UINTEGER:
			function = &ScalarFunction::UnaryFunction<uint32_t, TR, OP>;
			break;
		case LogicalTypeId::UBIGINT:
			function = &ScalarFunction::UnaryFunction<uint64_t, TR, OP>;
			break;
		case LogicalTypeId::HUGEINT:
			function = &ScalarFunction::UnaryFunction<hugeint_t, TR, OP>;
			break;
		case LogicalTypeId::FLOAT:
			function = &ScalarFunction::UnaryFunction<float, TR, OP>;
			break;
		case LogicalTypeId::DOUBLE:
			function = &ScalarFunction::UnaryFunction<double, TR, OP>;
			break;
		default:
			throw InternalException("Unimplemented type for GetScalarUnaryFunctionFixedReturn");
		}
		return function;
	}
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/function/aggregate_function.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/function/aggregate_state.hpp
//
//
//===----------------------------------------------------------------------===//





//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/storage/statistics/node_statistics.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {

class NodeStatistics {
public:
	NodeStatistics() : has_estimated_cardinality(false), has_max_cardinality(false) {
	}
	explicit NodeStatistics(idx_t estimated_cardinality)
	    : has_estimated_cardinality(true), estimated_cardinality(estimated_cardinality), has_max_cardinality(false) {
	}
	NodeStatistics(idx_t estimated_cardinality, idx_t max_cardinality)
	    : has_estimated_cardinality(true), estimated_cardinality(estimated_cardinality), has_max_cardinality(true),
	      max_cardinality(max_cardinality) {
	}
	void Serialize(Serializer &serializer) const {
		serializer.Write(has_estimated_cardinality);
		if (has_estimated_cardinality) {
			serializer.Write(estimated_cardinality);
			serializer.Write(has_max_cardinality);
			if (has_max_cardinality) {
				serializer.Write(max_cardinality);
			}
		} else {
			D_ASSERT(!has_max_cardinality);
		}
	}
	static unique_ptr<NodeStatistics> Deserialize(Deserializer &source) {
		bool has_estimated_cardinality = source.Read<bool>();
		if (!has_estimated_cardinality) {
			return make_uniq<NodeStatistics>();
		}
		idx_t estimated_cardinality = source.Read<idx_t>();
		bool has_max_cardinality = source.Read<bool>();
		if (!has_max_cardinality) {
			return make_uniq<NodeStatistics>(estimated_cardinality);
		}
		idx_t max_cardinality = source.Read<idx_t>();
		return make_uniq<NodeStatistics>(estimated_cardinality, max_cardinality);
	}

	//! Whether or not the node has an estimated cardinality specified
	bool has_estimated_cardinality;
	//! The estimated cardinality at the specified node
	idx_t estimated_cardinality;
	//! Whether or not the node has a maximum cardinality specified
	bool has_max_cardinality;
	//! The max possible cardinality at the specified node
	idx_t max_cardinality;
};

} // namespace duckdb


namespace duckdb {

enum class AggregateType : uint8_t { NON_DISTINCT = 1, DISTINCT = 2 };
//! Whether or not the input order influences the result of the aggregate
enum class AggregateOrderDependent : uint8_t { ORDER_DEPENDENT = 1, NOT_ORDER_DEPENDENT = 2 };

class BoundAggregateExpression;

struct AggregateInputData {
	AggregateInputData(optional_ptr<FunctionData> bind_data_p, Allocator &allocator_p)
	    : bind_data(bind_data_p), allocator(allocator_p) {
	}
	optional_ptr<FunctionData> bind_data;
	Allocator &allocator;
};

struct AggregateUnaryInput {
	AggregateUnaryInput(AggregateInputData &input_p, ValidityMask &input_mask_p)
	    : input(input_p), input_mask(input_mask_p), input_idx(0) {
	}

	AggregateInputData &input;
	ValidityMask &input_mask;
	idx_t input_idx;

	inline bool RowIsValid() {
		return input_mask.RowIsValid(input_idx);
	}
};

struct AggregateBinaryInput {
	AggregateBinaryInput(AggregateInputData &input_p, ValidityMask &left_mask_p, ValidityMask &right_mask_p)
	    : input(input_p), left_mask(left_mask_p), right_mask(right_mask_p) {
	}

	AggregateInputData &input;
	ValidityMask &left_mask;
	ValidityMask &right_mask;
	idx_t lidx;
	idx_t ridx;
};

struct AggregateFinalizeData {
	AggregateFinalizeData(Vector &result_p, AggregateInputData &input_p)
	    : result(result_p), input(input_p), result_idx(0) {
	}

	Vector &result;
	AggregateInputData &input;
	idx_t result_idx;

	inline void ReturnNull() {
		switch (result.GetVectorType()) {
		case VectorType::FLAT_VECTOR:
			FlatVector::SetNull(result, result_idx, true);
			break;
		case VectorType::CONSTANT_VECTOR:
			ConstantVector::SetNull(result, true);
			break;
		default:
			throw InternalException("Invalid result vector type for aggregate");
		}
	}

	inline string_t ReturnString(string_t value) {
		return StringVector::AddStringOrBlob(result, value);
	}
};

struct AggregateStatisticsInput {
	AggregateStatisticsInput(optional_ptr<FunctionData> bind_data_p, vector<BaseStatistics> &child_stats_p,
	                         optional_ptr<NodeStatistics> node_stats_p)
	    : bind_data(bind_data_p), child_stats(child_stats_p), node_stats(node_stats_p) {
	}

	optional_ptr<FunctionData> bind_data;
	vector<BaseStatistics> &child_stats;
	optional_ptr<NodeStatistics> node_stats;
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/planner/bound_result_modifier.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/result_modifier.hpp
//
//
//===----------------------------------------------------------------------===//





//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/order_type.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {

enum class OrderType : uint8_t { INVALID = 0, ORDER_DEFAULT = 1, ASCENDING = 2, DESCENDING = 3 };

enum class OrderByNullType : uint8_t { INVALID = 0, ORDER_DEFAULT = 1, NULLS_FIRST = 2, NULLS_LAST = 3 };

enum class DefaultOrderByNullType : uint8_t {
	INVALID = 0,
	NULLS_FIRST = 2,
	NULLS_LAST = 3,
	NULLS_FIRST_ON_ASC_LAST_ON_DESC = 4,
	NULLS_LAST_ON_ASC_FIRST_ON_DESC = 5
};

} // namespace duckdb



namespace duckdb {
class FieldWriter;
class FieldReader;
class FormatDeserializer;
class FormatSerializer;

enum class ResultModifierType : uint8_t {
	LIMIT_MODIFIER = 1,
	ORDER_MODIFIER = 2,
	DISTINCT_MODIFIER = 3,
	LIMIT_PERCENT_MODIFIER = 4
};

const char *ToString(ResultModifierType value);
ResultModifierType ResultModifierFromString(const char *value);

//! A ResultModifier
class ResultModifier {
public:
	explicit ResultModifier(ResultModifierType type) : type(type) {
	}
	virtual ~ResultModifier() {
	}

	ResultModifierType type;

public:
	//! Returns true if the two result modifiers are equivalent
	virtual bool Equals(const ResultModifier &other) const;

	//! Create a copy of this ResultModifier
	virtual unique_ptr<ResultModifier> Copy() const = 0;
	//! Serializes a ResultModifier to a stand-alone binary blob
	void Serialize(Serializer &serializer) const;
	//! Serializes a ResultModifier to a stand-alone binary blob
	virtual void Serialize(FieldWriter &writer) const = 0;
	//! Deserializes a blob back into a ResultModifier
	static unique_ptr<ResultModifier> Deserialize(Deserializer &source);

	virtual void FormatSerialize(FormatSerializer &serializer) const;
	static unique_ptr<ResultModifier> FormatDeserialize(FormatDeserializer &deserializer);

public:
	template <class TARGET>
	TARGET &Cast() {
		if (type != TARGET::TYPE) {
			throw InternalException("Failed to cast result modifier to type - result modifier type mismatch");
		}
		return reinterpret_cast<TARGET &>(*this);
	}

	template <class TARGET>
	const TARGET &Cast() const {
		if (type != TARGET::TYPE) {
			throw InternalException("Failed to cast result modifier to type - result modifier type mismatch");
		}
		return reinterpret_cast<const TARGET &>(*this);
	}
};

//! Single node in ORDER BY statement
struct OrderByNode {
	OrderByNode(OrderType type, OrderByNullType null_order, unique_ptr<ParsedExpression> expression)
	    : type(type), null_order(null_order), expression(std::move(expression)) {
	}

	//! Sort order, ASC or DESC
	OrderType type;
	//! The NULL sort order, NULLS_FIRST or NULLS_LAST
	OrderByNullType null_order;
	//! Expression to order by
	unique_ptr<ParsedExpression> expression;

public:
	void Serialize(Serializer &serializer) const;
	string ToString() const;
	static OrderByNode Deserialize(Deserializer &source);

	void FormatSerialize(FormatSerializer &serializer) const;
	static OrderByNode FormatDeserialize(FormatDeserializer &deserializer);
};

class LimitModifier : public ResultModifier {
public:
	static constexpr const ResultModifierType TYPE = ResultModifierType::LIMIT_MODIFIER;

public:
	LimitModifier() : ResultModifier(ResultModifierType::LIMIT_MODIFIER) {
	}

	//! LIMIT count
	unique_ptr<ParsedExpression> limit;
	//! OFFSET
	unique_ptr<ParsedExpression> offset;

public:
	bool Equals(const ResultModifier &other) const override;
	unique_ptr<ResultModifier> Copy() const override;
	void Serialize(FieldWriter &writer) const override;
	static unique_ptr<ResultModifier> Deserialize(FieldReader &reader);

	void FormatSerialize(FormatSerializer &serializer) const override;
	static unique_ptr<ResultModifier> FormatDeserialize(FormatDeserializer &deserializer);
};

class OrderModifier : public ResultModifier {
public:
	static constexpr const ResultModifierType TYPE = ResultModifierType::ORDER_MODIFIER;

public:
	OrderModifier() : ResultModifier(ResultModifierType::ORDER_MODIFIER) {
	}

	//! List of order nodes
	vector<OrderByNode> orders;

public:
	bool Equals(const ResultModifier &other) const override;
	unique_ptr<ResultModifier> Copy() const override;
	void Serialize(FieldWriter &writer) const override;
	static unique_ptr<ResultModifier> Deserialize(FieldReader &reader);

	void FormatSerialize(FormatSerializer &serializer) const override;
	static unique_ptr<ResultModifier> FormatDeserialize(FormatDeserializer &deserializer);

	static bool Equals(const unique_ptr<OrderModifier> &left, const unique_ptr<OrderModifier> &right);
};

class DistinctModifier : public ResultModifier {
public:
	static constexpr const ResultModifierType TYPE = ResultModifierType::DISTINCT_MODIFIER;

public:
	DistinctModifier() : ResultModifier(ResultModifierType::DISTINCT_MODIFIER) {
	}

	//! list of distinct on targets (if any)
	vector<unique_ptr<ParsedExpression>> distinct_on_targets;

public:
	bool Equals(const ResultModifier &other) const override;
	unique_ptr<ResultModifier> Copy() const override;
	void Serialize(FieldWriter &writer) const override;
	static unique_ptr<ResultModifier> Deserialize(FieldReader &reader);

	void FormatSerialize(FormatSerializer &serializer) const override;
	static unique_ptr<ResultModifier> FormatDeserialize(FormatDeserializer &deserializer);
};

class LimitPercentModifier : public ResultModifier {
public:
	static constexpr const ResultModifierType TYPE = ResultModifierType::LIMIT_PERCENT_MODIFIER;

public:
	LimitPercentModifier() : ResultModifier(ResultModifierType::LIMIT_PERCENT_MODIFIER) {
	}

	//! LIMIT %
	unique_ptr<ParsedExpression> limit;
	//! OFFSET
	unique_ptr<ParsedExpression> offset;

public:
	bool Equals(const ResultModifier &other) const override;
	unique_ptr<ResultModifier> Copy() const override;
	void Serialize(FieldWriter &writer) const override;
	static unique_ptr<ResultModifier> Deserialize(FieldReader &reader);

	void FormatSerialize(FormatSerializer &serializer) const override;
	static unique_ptr<ResultModifier> FormatDeserialize(FormatDeserializer &deserializer);
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/planner/bound_statement.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {

class LogicalOperator;
struct LogicalType;

struct BoundStatement {
	unique_ptr<LogicalOperator> plan;
	vector<LogicalType> types;
	vector<string> names;
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/planner/expression.hpp
//
//
//===----------------------------------------------------------------------===//







namespace duckdb {
class BaseStatistics;
class FieldWriter;
class FieldReader;
class ClientContext;

//!  The Expression class represents a bound Expression with a return type
class Expression : public BaseExpression {
public:
	Expression(ExpressionType type, ExpressionClass expression_class, LogicalType return_type);
	~Expression() override;

	//! The return type of the expression
	LogicalType return_type;
	//! Expression statistics (if any) - ONLY USED FOR VERIFICATION
	unique_ptr<BaseStatistics> verification_stats;

public:
	bool IsAggregate() const override;
	bool IsWindow() const override;
	bool HasSubquery() const override;
	bool IsScalar() const override;
	bool HasParameter() const override;
	virtual bool HasSideEffects() const;
	virtual bool PropagatesNullValues() const;
	virtual bool IsFoldable() const;

	hash_t Hash() const override;

	bool Equals(const BaseExpression &other) const override {
		if (!BaseExpression::Equals(other)) {
			return false;
		}
		return return_type == ((Expression &)other).return_type;
	}
	static bool Equals(const Expression &left, const Expression &right) {
		return left.Equals(right);
	}
	static bool Equals(const unique_ptr<Expression> &left, const unique_ptr<Expression> &right);
	static bool ListEquals(const vector<unique_ptr<Expression>> &left, const vector<unique_ptr<Expression>> &right);
	//! Create a copy of this expression
	virtual unique_ptr<Expression> Copy() = 0;

	//! Serializes an Expression to a stand-alone binary blob
	void Serialize(Serializer &serializer) const;
	//! Serializes an Expression to a stand-alone binary blob
	virtual void Serialize(FieldWriter &writer) const = 0;

	//! Deserializes a blob back into an Expression [CAN THROW:
	//! SerializationException]
	static unique_ptr<Expression> Deserialize(Deserializer &source, PlanDeserializationState &state);

protected:
	//! Copy base Expression properties from another expression to this one,
	//! used in Copy method
	void CopyProperties(Expression &other) {
		type = other.type;
		expression_class = other.expression_class;
		alias = other.alias;
		return_type = other.return_type;
	}
};

} // namespace duckdb



namespace duckdb {

//! A ResultModifier
class BoundResultModifier {
public:
	explicit BoundResultModifier(ResultModifierType type);
	virtual ~BoundResultModifier();

	ResultModifierType type;

public:
	template <class TARGET>
	TARGET &Cast() {
		if (type != TARGET::TYPE) {
			throw InternalException("Failed to cast result modifier to type - result modifier type mismatch");
		}
		return reinterpret_cast<TARGET &>(*this);
	}

	template <class TARGET>
	const TARGET &Cast() const {
		if (type != TARGET::TYPE) {
			throw InternalException("Failed to cast result modifier to type - result modifier type mismatch");
		}
		return reinterpret_cast<const TARGET &>(*this);
	}
};

struct BoundOrderByNode {
public:
	static constexpr const ResultModifierType TYPE = ResultModifierType::ORDER_MODIFIER;

public:
	BoundOrderByNode(OrderType type, OrderByNullType null_order, unique_ptr<Expression> expression);
	BoundOrderByNode(OrderType type, OrderByNullType null_order, unique_ptr<Expression> expression,
	                 unique_ptr<BaseStatistics> stats);

	OrderType type;
	OrderByNullType null_order;
	unique_ptr<Expression> expression;
	unique_ptr<BaseStatistics> stats;

public:
	BoundOrderByNode Copy() const;
	bool Equals(const BoundOrderByNode &other) const;
	string ToString() const;

	void Serialize(Serializer &serializer) const;
	static BoundOrderByNode Deserialize(Deserializer &source, PlanDeserializationState &state);
};

class BoundLimitModifier : public BoundResultModifier {
public:
	static constexpr const ResultModifierType TYPE = ResultModifierType::LIMIT_MODIFIER;

public:
	BoundLimitModifier();

	//! LIMIT
	int64_t limit_val = NumericLimits<int64_t>::Maximum();
	//! OFFSET
	int64_t offset_val = 0;
	//! Expression in case limit is not constant
	unique_ptr<Expression> limit;
	//! Expression in case limit is not constant
	unique_ptr<Expression> offset;
};

class BoundOrderModifier : public BoundResultModifier {
public:
	static constexpr const ResultModifierType TYPE = ResultModifierType::ORDER_MODIFIER;

public:
	BoundOrderModifier();

	//! List of order nodes
	vector<BoundOrderByNode> orders;

	unique_ptr<BoundOrderModifier> Copy() const;
	static bool Equals(const BoundOrderModifier &left, const BoundOrderModifier &right);
	static bool Equals(const unique_ptr<BoundOrderModifier> &left, const unique_ptr<BoundOrderModifier> &right);

	void Serialize(Serializer &serializer) const;
	static unique_ptr<BoundOrderModifier> Deserialize(Deserializer &source, PlanDeserializationState &state);
};

enum class DistinctType : uint8_t { DISTINCT = 0, DISTINCT_ON = 1 };

class BoundDistinctModifier : public BoundResultModifier {
public:
	static constexpr const ResultModifierType TYPE = ResultModifierType::DISTINCT_MODIFIER;

public:
	BoundDistinctModifier();

	//! Whether or not this is a DISTINCT or DISTINCT ON
	DistinctType distinct_type;
	//! list of distinct on targets
	vector<unique_ptr<Expression>> target_distincts;
};

class BoundLimitPercentModifier : public BoundResultModifier {
public:
	static constexpr const ResultModifierType TYPE = ResultModifierType::LIMIT_PERCENT_MODIFIER;

public:
	BoundLimitPercentModifier();

	//! LIMIT %
	double limit_percent = 100.0;
	//! OFFSET
	int64_t offset_val = 0;
	//! Expression in case limit is not constant
	unique_ptr<Expression> limit;
	//! Expression in case limit is not constant
	unique_ptr<Expression> offset;
};

} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/vector_operations/aggregate_executor.hpp
//
//
//===----------------------------------------------------------------------===//








namespace duckdb {

struct AggregateInputData;

typedef std::pair<idx_t, idx_t> FrameBounds;

class AggregateExecutor {
private:
	template <class STATE_TYPE, class OP>
	static inline void NullaryFlatLoop(STATE_TYPE **__restrict states, AggregateInputData &aggr_input_data,
	                                   idx_t count) {
		for (idx_t i = 0; i < count; i++) {
			OP::template Operation<STATE_TYPE, OP>(*states[i], aggr_input_data, i);
		}
	}

	template <class STATE_TYPE, class OP>
	static inline void NullaryScatterLoop(STATE_TYPE **__restrict states, AggregateInputData &aggr_input_data,
	                                      const SelectionVector &ssel, idx_t count) {

		for (idx_t i = 0; i < count; i++) {
			auto sidx = ssel.get_index(i);
			OP::template Operation<STATE_TYPE, OP>(*states[sidx], aggr_input_data, sidx);
		}
	}

	template <class STATE_TYPE, class INPUT_TYPE, class OP>
	static inline void UnaryFlatLoop(const INPUT_TYPE *__restrict idata, AggregateInputData &aggr_input_data,
	                                 STATE_TYPE **__restrict states, ValidityMask &mask, idx_t count) {
		if (OP::IgnoreNull() && !mask.AllValid()) {
			AggregateUnaryInput input(aggr_input_data, mask);
			auto &base_idx = input.input_idx;
			base_idx = 0;
			auto entry_count = ValidityMask::EntryCount(count);
			for (idx_t entry_idx = 0; entry_idx < entry_count; entry_idx++) {
				auto validity_entry = mask.GetValidityEntry(entry_idx);
				idx_t next = MinValue<idx_t>(base_idx + ValidityMask::BITS_PER_VALUE, count);
				if (ValidityMask::AllValid(validity_entry)) {
					// all valid: perform operation
					for (; base_idx < next; base_idx++) {
						OP::template Operation<INPUT_TYPE, STATE_TYPE, OP>(*states[base_idx], idata[base_idx], input);
					}
				} else if (ValidityMask::NoneValid(validity_entry)) {
					// nothing valid: skip all
					base_idx = next;
					continue;
				} else {
					// partially valid: need to check individual elements for validity
					idx_t start = base_idx;
					for (; base_idx < next; base_idx++) {
						if (ValidityMask::RowIsValid(validity_entry, base_idx - start)) {
							OP::template Operation<INPUT_TYPE, STATE_TYPE, OP>(*states[base_idx], idata[base_idx],
							                                                   input);
						}
					}
				}
			}
		} else {
			AggregateUnaryInput input(aggr_input_data, mask);
			auto &i = input.input_idx;
			for (i = 0; i < count; i++) {
				OP::template Operation<INPUT_TYPE, STATE_TYPE, OP>(*states[i], idata[i], input);
			}
		}
	}

	template <class STATE_TYPE, class INPUT_TYPE, class OP>
	static inline void UnaryScatterLoop(const INPUT_TYPE *__restrict idata, AggregateInputData &aggr_input_data,
	                                    STATE_TYPE **__restrict states, const SelectionVector &isel,
	                                    const SelectionVector &ssel, ValidityMask &mask, idx_t count) {
		if (OP::IgnoreNull() && !mask.AllValid()) {
			// potential NULL values and NULL values are ignored
			AggregateUnaryInput input(aggr_input_data, mask);
			for (idx_t i = 0; i < count; i++) {
				input.input_idx = isel.get_index(i);
				auto sidx = ssel.get_index(i);
				if (mask.RowIsValid(input.input_idx)) {
					OP::template Operation<INPUT_TYPE, STATE_TYPE, OP>(*states[sidx], idata[input.input_idx], input);
				}
			}
		} else {
			// quick path: no NULL values or NULL values are not ignored
			AggregateUnaryInput input(aggr_input_data, mask);
			for (idx_t i = 0; i < count; i++) {
				input.input_idx = isel.get_index(i);
				auto sidx = ssel.get_index(i);
				OP::template Operation<INPUT_TYPE, STATE_TYPE, OP>(*states[sidx], idata[input.input_idx], input);
			}
		}
	}

	template <class STATE_TYPE, class INPUT_TYPE, class OP>
	static inline void UnaryFlatUpdateLoop(const INPUT_TYPE *__restrict idata, AggregateInputData &aggr_input_data,
	                                       STATE_TYPE *__restrict state, idx_t count, ValidityMask &mask) {
		AggregateUnaryInput input(aggr_input_data, mask);
		auto &base_idx = input.input_idx;
		base_idx = 0;
		auto entry_count = ValidityMask::EntryCount(count);
		for (idx_t entry_idx = 0; entry_idx < entry_count; entry_idx++) {
			auto validity_entry = mask.GetValidityEntry(entry_idx);
			idx_t next = MinValue<idx_t>(base_idx + ValidityMask::BITS_PER_VALUE, count);
			if (!OP::IgnoreNull() || ValidityMask::AllValid(validity_entry)) {
				// all valid: perform operation
				for (; base_idx < next; base_idx++) {
					OP::template Operation<INPUT_TYPE, STATE_TYPE, OP>(*state, idata[base_idx], input);
				}
			} else if (ValidityMask::NoneValid(validity_entry)) {
				// nothing valid: skip all
				base_idx = next;
				continue;
			} else {
				// partially valid: need to check individual elements for validity
				idx_t start = base_idx;
				for (; base_idx < next; base_idx++) {
					if (ValidityMask::RowIsValid(validity_entry, base_idx - start)) {
						OP::template Operation<INPUT_TYPE, STATE_TYPE, OP>(*state, idata[base_idx], input);
					}
				}
			}
		}
	}

	template <class STATE_TYPE, class INPUT_TYPE, class OP>
	static inline void UnaryUpdateLoop(const INPUT_TYPE *__restrict idata, AggregateInputData &aggr_input_data,
	                                   STATE_TYPE *__restrict state, idx_t count, ValidityMask &mask,
	                                   const SelectionVector &__restrict sel_vector) {
		AggregateUnaryInput input(aggr_input_data, mask);
		if (OP::IgnoreNull() && !mask.AllValid()) {
			// potential NULL values and NULL values are ignored
			for (idx_t i = 0; i < count; i++) {
				input.input_idx = sel_vector.get_index(i);
				if (mask.RowIsValid(input.input_idx)) {
					OP::template Operation<INPUT_TYPE, STATE_TYPE, OP>(*state, idata[input.input_idx], input);
				}
			}
		} else {
			// quick path: no NULL values or NULL values are not ignored
			for (idx_t i = 0; i < count; i++) {
				input.input_idx = sel_vector.get_index(i);
				OP::template Operation<INPUT_TYPE, STATE_TYPE, OP>(*state, idata[input.input_idx], input);
			}
		}
	}

	template <class STATE_TYPE, class A_TYPE, class B_TYPE, class OP>
	static inline void BinaryScatterLoop(const A_TYPE *__restrict adata, AggregateInputData &aggr_input_data,
	                                     const B_TYPE *__restrict bdata, STATE_TYPE **__restrict states, idx_t count,
	                                     const SelectionVector &asel, const SelectionVector &bsel,
	                                     const SelectionVector &ssel, ValidityMask &avalidity,
	                                     ValidityMask &bvalidity) {
		AggregateBinaryInput input(aggr_input_data, avalidity, bvalidity);
		if (OP::IgnoreNull() && (!avalidity.AllValid() || !bvalidity.AllValid())) {
			// potential NULL values and NULL values are ignored
			for (idx_t i = 0; i < count; i++) {
				input.lidx = asel.get_index(i);
				input.ridx = bsel.get_index(i);
				auto sidx = ssel.get_index(i);
				if (avalidity.RowIsValid(input.lidx) && bvalidity.RowIsValid(input.ridx)) {
					OP::template Operation<A_TYPE, B_TYPE, STATE_TYPE, OP>(*states[sidx], adata[input.lidx],
					                                                       bdata[input.ridx], input);
				}
			}
		} else {
			// quick path: no NULL values or NULL values are not ignored
			for (idx_t i = 0; i < count; i++) {
				input.lidx = asel.get_index(i);
				input.ridx = bsel.get_index(i);
				auto sidx = ssel.get_index(i);
				OP::template Operation<A_TYPE, B_TYPE, STATE_TYPE, OP>(*states[sidx], adata[input.lidx],
				                                                       bdata[input.ridx], input);
			}
		}
	}

	template <class STATE_TYPE, class A_TYPE, class B_TYPE, class OP>
	static inline void BinaryUpdateLoop(const A_TYPE *__restrict adata, AggregateInputData &aggr_input_data,
	                                    const B_TYPE *__restrict bdata, STATE_TYPE *__restrict state, idx_t count,
	                                    const SelectionVector &asel, const SelectionVector &bsel,
	                                    ValidityMask &avalidity, ValidityMask &bvalidity) {
		AggregateBinaryInput input(aggr_input_data, avalidity, bvalidity);
		if (OP::IgnoreNull() && (!avalidity.AllValid() || !bvalidity.AllValid())) {
			// potential NULL values and NULL values are ignored
			for (idx_t i = 0; i < count; i++) {
				input.lidx = asel.get_index(i);
				input.ridx = bsel.get_index(i);
				if (avalidity.RowIsValid(input.lidx) && bvalidity.RowIsValid(input.ridx)) {
					OP::template Operation<A_TYPE, B_TYPE, STATE_TYPE, OP>(*state, adata[input.lidx], bdata[input.ridx],
					                                                       input);
				}
			}
		} else {
			// quick path: no NULL values or NULL values are not ignored
			for (idx_t i = 0; i < count; i++) {
				input.lidx = asel.get_index(i);
				input.ridx = bsel.get_index(i);
				OP::template Operation<A_TYPE, B_TYPE, STATE_TYPE, OP>(*state, adata[input.lidx], bdata[input.ridx],
				                                                       input);
			}
		}
	}

public:
	template <class STATE_TYPE, class OP>
	static void NullaryScatter(Vector &states, AggregateInputData &aggr_input_data, idx_t count) {
		if (states.GetVectorType() == VectorType::CONSTANT_VECTOR) {
			auto sdata = ConstantVector::GetData<STATE_TYPE *>(states);
			OP::template ConstantOperation<STATE_TYPE, OP>(**sdata, aggr_input_data, count);
		} else if (states.GetVectorType() == VectorType::FLAT_VECTOR) {
			auto sdata = FlatVector::GetData<STATE_TYPE *>(states);
			NullaryFlatLoop<STATE_TYPE, OP>(sdata, aggr_input_data, count);
		} else {
			UnifiedVectorFormat sdata;
			states.ToUnifiedFormat(count, sdata);
			NullaryScatterLoop<STATE_TYPE, OP>((STATE_TYPE **)sdata.data, aggr_input_data, *sdata.sel, count);
		}
	}

	template <class STATE_TYPE, class OP>
	static void NullaryUpdate(data_ptr_t state, AggregateInputData &aggr_input_data, idx_t count) {
		OP::template ConstantOperation<STATE_TYPE, OP>(*reinterpret_cast<STATE_TYPE *>(state), aggr_input_data, count);
	}

	template <class STATE_TYPE, class INPUT_TYPE, class OP>
	static void UnaryScatter(Vector &input, Vector &states, AggregateInputData &aggr_input_data, idx_t count) {
		if (input.GetVectorType() == VectorType::CONSTANT_VECTOR &&
		    states.GetVectorType() == VectorType::CONSTANT_VECTOR) {
			if (OP::IgnoreNull() && ConstantVector::IsNull(input)) {
				// constant NULL input in function that ignores NULL values
				return;
			}
			// regular constant: get first state
			auto idata = ConstantVector::GetData<INPUT_TYPE>(input);
			auto sdata = ConstantVector::GetData<STATE_TYPE *>(states);
			AggregateUnaryInput input_data(aggr_input_data, ConstantVector::Validity(input));
			OP::template ConstantOperation<INPUT_TYPE, STATE_TYPE, OP>(**sdata, *idata, input_data, count);
		} else if (input.GetVectorType() == VectorType::FLAT_VECTOR &&
		           states.GetVectorType() == VectorType::FLAT_VECTOR) {
			auto idata = FlatVector::GetData<INPUT_TYPE>(input);
			auto sdata = FlatVector::GetData<STATE_TYPE *>(states);
			UnaryFlatLoop<STATE_TYPE, INPUT_TYPE, OP>(idata, aggr_input_data, sdata, FlatVector::Validity(input),
			                                          count);
		} else {
			UnifiedVectorFormat idata, sdata;
			input.ToUnifiedFormat(count, idata);
			states.ToUnifiedFormat(count, sdata);
			UnaryScatterLoop<STATE_TYPE, INPUT_TYPE, OP>(UnifiedVectorFormat::GetData<INPUT_TYPE>(idata),
			                                             aggr_input_data, (STATE_TYPE **)sdata.data, *idata.sel,
			                                             *sdata.sel, idata.validity, count);
		}
	}

	template <class STATE_TYPE, class INPUT_TYPE, class OP>
	static void UnaryUpdate(Vector &input, AggregateInputData &aggr_input_data, data_ptr_t state, idx_t count) {
		switch (input.GetVectorType()) {
		case VectorType::CONSTANT_VECTOR: {
			if (OP::IgnoreNull() && ConstantVector::IsNull(input)) {
				return;
			}
			auto idata = ConstantVector::GetData<INPUT_TYPE>(input);
			AggregateUnaryInput input_data(aggr_input_data, ConstantVector::Validity(input));
			OP::template ConstantOperation<INPUT_TYPE, STATE_TYPE, OP>(*reinterpret_cast<STATE_TYPE *>(state), *idata,
			                                                           input_data, count);
			break;
		}
		case VectorType::FLAT_VECTOR: {
			auto idata = FlatVector::GetData<INPUT_TYPE>(input);
			UnaryFlatUpdateLoop<STATE_TYPE, INPUT_TYPE, OP>(idata, aggr_input_data, (STATE_TYPE *)state, count,
			                                                FlatVector::Validity(input));
			break;
		}
		default: {
			UnifiedVectorFormat idata;
			input.ToUnifiedFormat(count, idata);
			UnaryUpdateLoop<STATE_TYPE, INPUT_TYPE, OP>(UnifiedVectorFormat::GetData<INPUT_TYPE>(idata),
			                                            aggr_input_data, (STATE_TYPE *)state, count, idata.validity,
			                                            *idata.sel);
			break;
		}
		}
	}

	template <class STATE_TYPE, class A_TYPE, class B_TYPE, class OP>
	static void BinaryScatter(AggregateInputData &aggr_input_data, Vector &a, Vector &b, Vector &states, idx_t count) {
		UnifiedVectorFormat adata, bdata, sdata;

		a.ToUnifiedFormat(count, adata);
		b.ToUnifiedFormat(count, bdata);
		states.ToUnifiedFormat(count, sdata);

		BinaryScatterLoop<STATE_TYPE, A_TYPE, B_TYPE, OP>(
		    UnifiedVectorFormat::GetData<A_TYPE>(adata), aggr_input_data, UnifiedVectorFormat::GetData<B_TYPE>(bdata),
		    (STATE_TYPE **)sdata.data, count, *adata.sel, *bdata.sel, *sdata.sel, adata.validity, bdata.validity);
	}

	template <class STATE_TYPE, class A_TYPE, class B_TYPE, class OP>
	static void BinaryUpdate(AggregateInputData &aggr_input_data, Vector &a, Vector &b, data_ptr_t state, idx_t count) {
		UnifiedVectorFormat adata, bdata;

		a.ToUnifiedFormat(count, adata);
		b.ToUnifiedFormat(count, bdata);

		BinaryUpdateLoop<STATE_TYPE, A_TYPE, B_TYPE, OP>(
		    UnifiedVectorFormat::GetData<A_TYPE>(adata), aggr_input_data, UnifiedVectorFormat::GetData<B_TYPE>(bdata),
		    (STATE_TYPE *)state, count, *adata.sel, *bdata.sel, adata.validity, bdata.validity);
	}

	template <class STATE_TYPE, class OP>
	static void Combine(Vector &source, Vector &target, AggregateInputData &aggr_input_data, idx_t count) {
		D_ASSERT(source.GetType().id() == LogicalTypeId::POINTER && target.GetType().id() == LogicalTypeId::POINTER);
		auto sdata = FlatVector::GetData<const STATE_TYPE *>(source);
		auto tdata = FlatVector::GetData<STATE_TYPE *>(target);

		for (idx_t i = 0; i < count; i++) {
			OP::template Combine<STATE_TYPE, OP>(*sdata[i], *tdata[i], aggr_input_data);
		}
	}

	template <class STATE_TYPE, class RESULT_TYPE, class OP>
	static void Finalize(Vector &states, AggregateInputData &aggr_input_data, Vector &result, idx_t count,
	                     idx_t offset) {
		if (states.GetVectorType() == VectorType::CONSTANT_VECTOR) {
			result.SetVectorType(VectorType::CONSTANT_VECTOR);

			auto sdata = ConstantVector::GetData<STATE_TYPE *>(states);
			auto rdata = ConstantVector::GetData<RESULT_TYPE>(result);
			AggregateFinalizeData finalize_data(result, aggr_input_data);
			OP::template Finalize<RESULT_TYPE, STATE_TYPE>(**sdata, *rdata, finalize_data);
		} else {
			D_ASSERT(states.GetVectorType() == VectorType::FLAT_VECTOR);
			result.SetVectorType(VectorType::FLAT_VECTOR);

			auto sdata = FlatVector::GetData<STATE_TYPE *>(states);
			auto rdata = FlatVector::GetData<RESULT_TYPE>(result);
			AggregateFinalizeData finalize_data(result, aggr_input_data);
			for (idx_t i = 0; i < count; i++) {
				finalize_data.result_idx = i + offset;
				OP::template Finalize<RESULT_TYPE, STATE_TYPE>(*sdata[i], rdata[finalize_data.result_idx],
				                                               finalize_data);
			}
		}
	}

	template <class STATE_TYPE, class OP>
	static void VoidFinalize(Vector &states, AggregateInputData &aggr_input_data, Vector &result, idx_t count,
	                         idx_t offset) {
		if (states.GetVectorType() == VectorType::CONSTANT_VECTOR) {
			result.SetVectorType(VectorType::CONSTANT_VECTOR);

			auto sdata = ConstantVector::GetData<STATE_TYPE *>(states);
			AggregateFinalizeData finalize_data(result, aggr_input_data);
			OP::template Finalize<STATE_TYPE>(**sdata, finalize_data);
		} else {
			D_ASSERT(states.GetVectorType() == VectorType::FLAT_VECTOR);
			result.SetVectorType(VectorType::FLAT_VECTOR);

			auto sdata = FlatVector::GetData<STATE_TYPE *>(states);
			AggregateFinalizeData finalize_data(result, aggr_input_data);
			for (idx_t i = 0; i < count; i++) {
				finalize_data.result_idx = i + offset;
				OP::template Finalize<STATE_TYPE>(*sdata[i], finalize_data);
			}
		}
	}

	template <class STATE, class INPUT_TYPE, class RESULT_TYPE, class OP>
	static void UnaryWindow(Vector &input, const ValidityMask &ifilter, AggregateInputData &aggr_input_data,
	                        data_ptr_t state, const FrameBounds &frame, const FrameBounds &prev, Vector &result,
	                        idx_t rid, idx_t bias) {

		auto idata = FlatVector::GetData<const INPUT_TYPE>(input) - bias;
		const auto &ivalid = FlatVector::Validity(input);
		OP::template Window<STATE, INPUT_TYPE, RESULT_TYPE>(
		    idata, ifilter, ivalid, aggr_input_data, *reinterpret_cast<STATE *>(state), frame, prev, result, rid, bias);
	}

	template <class STATE_TYPE, class OP>
	static void Destroy(Vector &states, AggregateInputData &aggr_input_data, idx_t count) {
		auto sdata = FlatVector::GetData<STATE_TYPE *>(states);
		for (idx_t i = 0; i < count; i++) {
			OP::template Destroy<STATE_TYPE>(*sdata[i], aggr_input_data);
		}
	}
};

} // namespace duckdb


namespace duckdb {

//! The type used for sizing hashed aggregate function states
typedef idx_t (*aggregate_size_t)();
//! The type used for initializing hashed aggregate function states
typedef void (*aggregate_initialize_t)(data_ptr_t state);
//! The type used for updating hashed aggregate functions
typedef void (*aggregate_update_t)(Vector inputs[], AggregateInputData &aggr_input_data, idx_t input_count,
                                   Vector &state, idx_t count);
//! The type used for combining hashed aggregate states
typedef void (*aggregate_combine_t)(Vector &state, Vector &combined, AggregateInputData &aggr_input_data, idx_t count);
//! The type used for finalizing hashed aggregate function payloads
typedef void (*aggregate_finalize_t)(Vector &state, AggregateInputData &aggr_input_data, Vector &result, idx_t count,
                                     idx_t offset);
//! The type used for propagating statistics in aggregate functions (optional)
typedef unique_ptr<BaseStatistics> (*aggregate_statistics_t)(ClientContext &context, BoundAggregateExpression &expr,
                                                             AggregateStatisticsInput &input);
//! Binds the scalar function and creates the function data
typedef unique_ptr<FunctionData> (*bind_aggregate_function_t)(ClientContext &context, AggregateFunction &function,
                                                              vector<unique_ptr<Expression>> &arguments);
//! The type used for the aggregate destructor method. NOTE: this method is used in destructors and MAY NOT throw.
typedef void (*aggregate_destructor_t)(Vector &state, AggregateInputData &aggr_input_data, idx_t count);

//! The type used for updating simple (non-grouped) aggregate functions
typedef void (*aggregate_simple_update_t)(Vector inputs[], AggregateInputData &aggr_input_data, idx_t input_count,
                                          data_ptr_t state, idx_t count);

//! The type used for updating complex windowed aggregate functions (optional)
typedef std::pair<idx_t, idx_t> FrameBounds;
typedef void (*aggregate_window_t)(Vector inputs[], const ValidityMask &filter_mask,
                                   AggregateInputData &aggr_input_data, idx_t input_count, data_ptr_t state,
                                   const FrameBounds &frame, const FrameBounds &prev, Vector &result, idx_t rid,
                                   idx_t bias);

typedef void (*aggregate_serialize_t)(FieldWriter &writer, const FunctionData *bind_data,
                                      const AggregateFunction &function);
typedef unique_ptr<FunctionData> (*aggregate_deserialize_t)(PlanDeserializationState &context, FieldReader &reader,
                                                            AggregateFunction &function);

class AggregateFunction : public BaseScalarFunction {
public:
	AggregateFunction(const string &name, const vector<LogicalType> &arguments, const LogicalType &return_type,
	                  aggregate_size_t state_size, aggregate_initialize_t initialize, aggregate_update_t update,
	                  aggregate_combine_t combine, aggregate_finalize_t finalize,
	                  FunctionNullHandling null_handling = FunctionNullHandling::DEFAULT_NULL_HANDLING,
	                  aggregate_simple_update_t simple_update = nullptr, bind_aggregate_function_t bind = nullptr,
	                  aggregate_destructor_t destructor = nullptr, aggregate_statistics_t statistics = nullptr,
	                  aggregate_window_t window = nullptr, aggregate_serialize_t serialize = nullptr,
	                  aggregate_deserialize_t deserialize = nullptr)
	    : BaseScalarFunction(name, arguments, return_type, FunctionSideEffects::NO_SIDE_EFFECTS,
	                         LogicalType(LogicalTypeId::INVALID), null_handling),
	      state_size(state_size), initialize(initialize), update(update), combine(combine), finalize(finalize),
	      simple_update(simple_update), window(window), bind(bind), destructor(destructor), statistics(statistics),
	      serialize(serialize), deserialize(deserialize), order_dependent(AggregateOrderDependent::ORDER_DEPENDENT) {
	}

	AggregateFunction(const string &name, const vector<LogicalType> &arguments, const LogicalType &return_type,
	                  aggregate_size_t state_size, aggregate_initialize_t initialize, aggregate_update_t update,
	                  aggregate_combine_t combine, aggregate_finalize_t finalize,
	                  aggregate_simple_update_t simple_update = nullptr, bind_aggregate_function_t bind = nullptr,
	                  aggregate_destructor_t destructor = nullptr, aggregate_statistics_t statistics = nullptr,
	                  aggregate_window_t window = nullptr, aggregate_serialize_t serialize = nullptr,
	                  aggregate_deserialize_t deserialize = nullptr)
	    : BaseScalarFunction(name, arguments, return_type, FunctionSideEffects::NO_SIDE_EFFECTS,
	                         LogicalType(LogicalTypeId::INVALID)),
	      state_size(state_size), initialize(initialize), update(update), combine(combine), finalize(finalize),
	      simple_update(simple_update), window(window), bind(bind), destructor(destructor), statistics(statistics),
	      serialize(serialize), deserialize(deserialize), order_dependent(AggregateOrderDependent::ORDER_DEPENDENT) {
	}

	AggregateFunction(const vector<LogicalType> &arguments, const LogicalType &return_type, aggregate_size_t state_size,
	                  aggregate_initialize_t initialize, aggregate_update_t update, aggregate_combine_t combine,
	                  aggregate_finalize_t finalize,
	                  FunctionNullHandling null_handling = FunctionNullHandling::DEFAULT_NULL_HANDLING,
	                  aggregate_simple_update_t simple_update = nullptr, bind_aggregate_function_t bind = nullptr,
	                  aggregate_destructor_t destructor = nullptr, aggregate_statistics_t statistics = nullptr,
	                  aggregate_window_t window = nullptr, aggregate_serialize_t serialize = nullptr,
	                  aggregate_deserialize_t deserialize = nullptr)
	    : AggregateFunction(string(), arguments, return_type, state_size, initialize, update, combine, finalize,
	                        null_handling, simple_update, bind, destructor, statistics, window, serialize,
	                        deserialize) {
	}

	AggregateFunction(const vector<LogicalType> &arguments, const LogicalType &return_type, aggregate_size_t state_size,
	                  aggregate_initialize_t initialize, aggregate_update_t update, aggregate_combine_t combine,
	                  aggregate_finalize_t finalize, aggregate_simple_update_t simple_update = nullptr,
	                  bind_aggregate_function_t bind = nullptr, aggregate_destructor_t destructor = nullptr,
	                  aggregate_statistics_t statistics = nullptr, aggregate_window_t window = nullptr,
	                  aggregate_serialize_t serialize = nullptr, aggregate_deserialize_t deserialize = nullptr)
	    : AggregateFunction(string(), arguments, return_type, state_size, initialize, update, combine, finalize,
	                        FunctionNullHandling::DEFAULT_NULL_HANDLING, simple_update, bind, destructor, statistics,
	                        window, serialize, deserialize) {
	}
	//! The hashed aggregate state sizing function
	aggregate_size_t state_size;
	//! The hashed aggregate state initialization function
	aggregate_initialize_t initialize;
	//! The hashed aggregate update state function
	aggregate_update_t update;
	//! The hashed aggregate combine states function
	aggregate_combine_t combine;
	//! The hashed aggregate finalization function
	aggregate_finalize_t finalize;
	//! The simple aggregate update function (may be null)
	aggregate_simple_update_t simple_update;
	//! The windowed aggregate frame update function (may be null)
	aggregate_window_t window;

	//! The bind function (may be null)
	bind_aggregate_function_t bind;
	//! The destructor method (may be null)
	aggregate_destructor_t destructor;

	//! The statistics propagation function (may be null)
	aggregate_statistics_t statistics;

	aggregate_serialize_t serialize;
	aggregate_deserialize_t deserialize;
	//! Whether or not the aggregate is order dependent
	AggregateOrderDependent order_dependent;

	bool operator==(const AggregateFunction &rhs) const {
		return state_size == rhs.state_size && initialize == rhs.initialize && update == rhs.update &&
		       combine == rhs.combine && finalize == rhs.finalize && window == rhs.window;
	}
	bool operator!=(const AggregateFunction &rhs) const {
		return !(*this == rhs);
	}

public:
	template <class STATE, class RESULT_TYPE, class OP>
	static AggregateFunction NullaryAggregate(LogicalType return_type) {
		return AggregateFunction(
		    {}, return_type, AggregateFunction::StateSize<STATE>, AggregateFunction::StateInitialize<STATE, OP>,
		    AggregateFunction::NullaryScatterUpdate<STATE, OP>, AggregateFunction::StateCombine<STATE, OP>,
		    AggregateFunction::StateFinalize<STATE, RESULT_TYPE, OP>, AggregateFunction::NullaryUpdate<STATE, OP>);
	}

	template <class STATE, class INPUT_TYPE, class RESULT_TYPE, class OP>
	static AggregateFunction
	UnaryAggregate(const LogicalType &input_type, LogicalType return_type,
	               FunctionNullHandling null_handling = FunctionNullHandling::DEFAULT_NULL_HANDLING) {
		return AggregateFunction(
		    {input_type}, return_type, AggregateFunction::StateSize<STATE>,
		    AggregateFunction::StateInitialize<STATE, OP>, AggregateFunction::UnaryScatterUpdate<STATE, INPUT_TYPE, OP>,
		    AggregateFunction::StateCombine<STATE, OP>, AggregateFunction::StateFinalize<STATE, RESULT_TYPE, OP>,
		    null_handling, AggregateFunction::UnaryUpdate<STATE, INPUT_TYPE, OP>);
	}

	template <class STATE, class INPUT_TYPE, class RESULT_TYPE, class OP>
	static AggregateFunction UnaryAggregateDestructor(LogicalType input_type, LogicalType return_type) {
		auto aggregate = UnaryAggregate<STATE, INPUT_TYPE, RESULT_TYPE, OP>(input_type, return_type);
		aggregate.destructor = AggregateFunction::StateDestroy<STATE, OP>;
		return aggregate;
	}

	template <class STATE, class A_TYPE, class B_TYPE, class RESULT_TYPE, class OP>
	static AggregateFunction BinaryAggregate(const LogicalType &a_type, const LogicalType &b_type,
	                                         LogicalType return_type) {
		return AggregateFunction({a_type, b_type}, return_type, AggregateFunction::StateSize<STATE>,
		                         AggregateFunction::StateInitialize<STATE, OP>,
		                         AggregateFunction::BinaryScatterUpdate<STATE, A_TYPE, B_TYPE, OP>,
		                         AggregateFunction::StateCombine<STATE, OP>,
		                         AggregateFunction::StateFinalize<STATE, RESULT_TYPE, OP>,
		                         AggregateFunction::BinaryUpdate<STATE, A_TYPE, B_TYPE, OP>);
	}

public:
	template <class STATE>
	static idx_t StateSize() {
		return sizeof(STATE);
	}

	template <class STATE, class OP>
	static void StateInitialize(data_ptr_t state) {
		OP::Initialize(*reinterpret_cast<STATE *>(state));
	}

	template <class STATE, class OP>
	static void NullaryScatterUpdate(Vector inputs[], AggregateInputData &aggr_input_data, idx_t input_count,
	                                 Vector &states, idx_t count) {
		D_ASSERT(input_count == 0);
		AggregateExecutor::NullaryScatter<STATE, OP>(states, aggr_input_data, count);
	}

	template <class STATE, class OP>
	static void NullaryUpdate(Vector inputs[], AggregateInputData &aggr_input_data, idx_t input_count, data_ptr_t state,
	                          idx_t count) {
		D_ASSERT(input_count == 0);
		AggregateExecutor::NullaryUpdate<STATE, OP>(state, aggr_input_data, count);
	}

	template <class STATE, class T, class OP>
	static void UnaryScatterUpdate(Vector inputs[], AggregateInputData &aggr_input_data, idx_t input_count,
	                               Vector &states, idx_t count) {
		D_ASSERT(input_count == 1);
		AggregateExecutor::UnaryScatter<STATE, T, OP>(inputs[0], states, aggr_input_data, count);
	}

	template <class STATE, class INPUT_TYPE, class OP>
	static void UnaryUpdate(Vector inputs[], AggregateInputData &aggr_input_data, idx_t input_count, data_ptr_t state,
	                        idx_t count) {
		D_ASSERT(input_count == 1);
		AggregateExecutor::UnaryUpdate<STATE, INPUT_TYPE, OP>(inputs[0], aggr_input_data, state, count);
	}

	template <class STATE, class INPUT_TYPE, class RESULT_TYPE, class OP>
	static void UnaryWindow(Vector inputs[], const ValidityMask &filter_mask, AggregateInputData &aggr_input_data,
	                        idx_t input_count, data_ptr_t state, const FrameBounds &frame, const FrameBounds &prev,
	                        Vector &result, idx_t rid, idx_t bias) {
		D_ASSERT(input_count == 1);
		AggregateExecutor::UnaryWindow<STATE, INPUT_TYPE, RESULT_TYPE, OP>(inputs[0], filter_mask, aggr_input_data,
		                                                                   state, frame, prev, result, rid, bias);
	}

	template <class STATE, class A_TYPE, class B_TYPE, class OP>
	static void BinaryScatterUpdate(Vector inputs[], AggregateInputData &aggr_input_data, idx_t input_count,
	                                Vector &states, idx_t count) {
		D_ASSERT(input_count == 2);
		AggregateExecutor::BinaryScatter<STATE, A_TYPE, B_TYPE, OP>(aggr_input_data, inputs[0], inputs[1], states,
		                                                            count);
	}

	template <class STATE, class A_TYPE, class B_TYPE, class OP>
	static void BinaryUpdate(Vector inputs[], AggregateInputData &aggr_input_data, idx_t input_count, data_ptr_t state,
	                         idx_t count) {
		D_ASSERT(input_count == 2);
		AggregateExecutor::BinaryUpdate<STATE, A_TYPE, B_TYPE, OP>(aggr_input_data, inputs[0], inputs[1], state, count);
	}

	template <class STATE, class OP>
	static void StateCombine(Vector &source, Vector &target, AggregateInputData &aggr_input_data, idx_t count) {
		AggregateExecutor::Combine<STATE, OP>(source, target, aggr_input_data, count);
	}

	template <class STATE, class RESULT_TYPE, class OP>
	static void StateFinalize(Vector &states, AggregateInputData &aggr_input_data, Vector &result, idx_t count,
	                          idx_t offset) {
		AggregateExecutor::Finalize<STATE, RESULT_TYPE, OP>(states, aggr_input_data, result, count, offset);
	}

	template <class STATE, class OP>
	static void StateVoidFinalize(Vector &states, AggregateInputData &aggr_input_data, Vector &result, idx_t count,
	                              idx_t offset) {
		AggregateExecutor::VoidFinalize<STATE, OP>(states, aggr_input_data, result, count, offset);
	}

	template <class STATE, class OP>
	static void StateDestroy(Vector &states, AggregateInputData &aggr_input_data, idx_t count) {
		AggregateExecutor::Destroy<STATE, OP>(states, aggr_input_data, count);
	}
};

} // namespace duckdb


namespace duckdb {

struct UDFWrapper {
public:
	template <typename TR, typename... Args>
	inline static scalar_function_t CreateScalarFunction(const string &name, TR (*udf_func)(Args...)) {
		const std::size_t num_template_argc = sizeof...(Args);
		switch (num_template_argc) {
		case 1:
			return CreateUnaryFunction<TR, Args...>(name, udf_func);
		case 2:
			return CreateBinaryFunction<TR, Args...>(name, udf_func);
		case 3:
			return CreateTernaryFunction<TR, Args...>(name, udf_func);
		default: // LCOV_EXCL_START
			throw std::runtime_error("UDF function only supported until ternary!");
		} // LCOV_EXCL_STOP
	}

	template <typename TR, typename... Args>
	inline static scalar_function_t CreateScalarFunction(const string &name, vector<LogicalType> args,
	                                                     LogicalType ret_type, TR (*udf_func)(Args...)) {
		if (!TypesMatch<TR>(ret_type)) { // LCOV_EXCL_START
			throw std::runtime_error("Return type doesn't match with the first template type.");
		} // LCOV_EXCL_STOP

		const std::size_t num_template_types = sizeof...(Args);
		if (num_template_types != args.size()) { // LCOV_EXCL_START
			throw std::runtime_error(
			    "The number of templated types should be the same quantity of the LogicalType arguments.");
		} // LCOV_EXCL_STOP

		switch (num_template_types) {
		case 1:
			return CreateUnaryFunction<TR, Args...>(name, args, ret_type, udf_func);
		case 2:
			return CreateBinaryFunction<TR, Args...>(name, args, ret_type, udf_func);
		case 3:
			return CreateTernaryFunction<TR, Args...>(name, args, ret_type, udf_func);
		default: // LCOV_EXCL_START
			throw std::runtime_error("UDF function only supported until ternary!");
		} // LCOV_EXCL_STOP
	}

	template <typename TR, typename... Args>
	inline static void RegisterFunction(const string &name, scalar_function_t udf_function, ClientContext &context,
	                                    LogicalType varargs = LogicalType(LogicalTypeId::INVALID)) {
		vector<LogicalType> arguments;
		GetArgumentTypesRecursive<Args...>(arguments);

		LogicalType ret_type = GetArgumentType<TR>();

		RegisterFunction(name, arguments, ret_type, udf_function, context, varargs);
	}

	static void RegisterFunction(string name, vector<LogicalType> args, LogicalType ret_type,
	                             scalar_function_t udf_function, ClientContext &context,
	                             LogicalType varargs = LogicalType(LogicalTypeId::INVALID));

	//--------------------------------- Aggregate UDFs ------------------------------------//
	template <typename UDF_OP, typename STATE, typename TR, typename TA>
	inline static AggregateFunction CreateAggregateFunction(const string &name) {
		return CreateUnaryAggregateFunction<UDF_OP, STATE, TR, TA>(name);
	}

	template <typename UDF_OP, typename STATE, typename TR, typename TA, typename TB>
	inline static AggregateFunction CreateAggregateFunction(const string &name) {
		return CreateBinaryAggregateFunction<UDF_OP, STATE, TR, TA, TB>(name);
	}

	template <typename UDF_OP, typename STATE, typename TR, typename TA>
	inline static AggregateFunction CreateAggregateFunction(const string &name, LogicalType ret_type,
	                                                        LogicalType input_type) {
		if (!TypesMatch<TR>(ret_type)) { // LCOV_EXCL_START
			throw std::runtime_error("The return argument don't match!");
		} // LCOV_EXCL_STOP

		if (!TypesMatch<TA>(input_type)) { // LCOV_EXCL_START
			throw std::runtime_error("The input argument don't match!");
		} // LCOV_EXCL_STOP

		return CreateUnaryAggregateFunction<UDF_OP, STATE, TR, TA>(name, ret_type, input_type);
	}

	template <typename UDF_OP, typename STATE, typename TR, typename TA, typename TB>
	inline static AggregateFunction CreateAggregateFunction(const string &name, LogicalType ret_type,
	                                                        LogicalType input_typeA, LogicalType input_typeB) {
		if (!TypesMatch<TR>(ret_type)) { // LCOV_EXCL_START
			throw std::runtime_error("The return argument don't match!");
		}

		if (!TypesMatch<TA>(input_typeA)) {
			throw std::runtime_error("The first input argument don't match!");
		}

		if (!TypesMatch<TB>(input_typeB)) {
			throw std::runtime_error("The second input argument don't match!");
		} // LCOV_EXCL_STOP

		return CreateBinaryAggregateFunction<UDF_OP, STATE, TR, TA, TB>(name, ret_type, input_typeA, input_typeB);
	}

	//! A generic CreateAggregateFunction ---------------------------------------------------------------------------//
	inline static AggregateFunction
	CreateAggregateFunction(string name, vector<LogicalType> arguments, LogicalType return_type,
	                        aggregate_size_t state_size, aggregate_initialize_t initialize, aggregate_update_t update,
	                        aggregate_combine_t combine, aggregate_finalize_t finalize,
	                        aggregate_simple_update_t simple_update = nullptr, bind_aggregate_function_t bind = nullptr,
	                        aggregate_destructor_t destructor = nullptr) {

		AggregateFunction aggr_function(std::move(name), std::move(arguments), std::move(return_type), state_size,
		                                initialize, update, combine, finalize, simple_update, bind, destructor);
		aggr_function.null_handling = FunctionNullHandling::SPECIAL_HANDLING;
		return aggr_function;
	}

	static void RegisterAggrFunction(AggregateFunction aggr_function, ClientContext &context,
	                                 LogicalType varargs = LogicalType(LogicalTypeId::INVALID));

private:
	//-------------------------------- Templated functions --------------------------------//
	struct UnaryUDFExecutor {
		template <class INPUT_TYPE, class RESULT_TYPE>
		static RESULT_TYPE Operation(INPUT_TYPE input, ValidityMask &mask, idx_t idx, void *dataptr) {
			typedef RESULT_TYPE (*unary_function_t)(INPUT_TYPE);
			auto udf = (unary_function_t)dataptr;
			return udf(input);
		}
	};

	template <typename TR, typename TA>
	inline static scalar_function_t CreateUnaryFunction(const string &name, TR (*udf_func)(TA)) {
		scalar_function_t udf_function = [=](DataChunk &input, ExpressionState &state, Vector &result) -> void {
			UnaryExecutor::GenericExecute<TA, TR, UnaryUDFExecutor>(input.data[0], result, input.size(),
			                                                        (void *)udf_func);
		};
		return udf_function;
	}

	template <typename TR, typename TA, typename TB>
	inline static scalar_function_t CreateBinaryFunction(const string &name, TR (*udf_func)(TA, TB)) {
		scalar_function_t udf_function = [=](DataChunk &input, ExpressionState &state, Vector &result) -> void {
			BinaryExecutor::Execute<TA, TB, TR>(input.data[0], input.data[1], result, input.size(), udf_func);
		};
		return udf_function;
	}

	template <typename TR, typename TA, typename TB, typename TC>
	inline static scalar_function_t CreateTernaryFunction(const string &name, TR (*udf_func)(TA, TB, TC)) {
		scalar_function_t udf_function = [=](DataChunk &input, ExpressionState &state, Vector &result) -> void {
			TernaryExecutor::Execute<TA, TB, TC, TR>(input.data[0], input.data[1], input.data[2], result, input.size(),
			                                         udf_func);
		};
		return udf_function;
	}

	template <typename TR, typename... Args>
	inline static scalar_function_t CreateUnaryFunction(const string &name,
	                                                    TR (*udf_func)(Args...)) { // LCOV_EXCL_START
		throw std::runtime_error("Incorrect number of arguments for unary function");
	} // LCOV_EXCL_STOP

	template <typename TR, typename... Args>
	inline static scalar_function_t CreateBinaryFunction(const string &name,
	                                                     TR (*udf_func)(Args...)) { // LCOV_EXCL_START
		throw std::runtime_error("Incorrect number of arguments for binary function");
	} // LCOV_EXCL_STOP

	template <typename TR, typename... Args>
	inline static scalar_function_t CreateTernaryFunction(const string &name,
	                                                      TR (*udf_func)(Args...)) { // LCOV_EXCL_START
		throw std::runtime_error("Incorrect number of arguments for ternary function");
	} // LCOV_EXCL_STOP

	template <typename T>
	inline static LogicalType GetArgumentType() {
		if (std::is_same<T, bool>()) {
			return LogicalType(LogicalTypeId::BOOLEAN);
		} else if (std::is_same<T, int8_t>()) {
			return LogicalType(LogicalTypeId::TINYINT);
		} else if (std::is_same<T, int16_t>()) {
			return LogicalType(LogicalTypeId::SMALLINT);
		} else if (std::is_same<T, int32_t>()) {
			return LogicalType(LogicalTypeId::INTEGER);
		} else if (std::is_same<T, int64_t>()) {
			return LogicalType(LogicalTypeId::BIGINT);
		} else if (std::is_same<T, float>()) {
			return LogicalType(LogicalTypeId::FLOAT);
		} else if (std::is_same<T, double>()) {
			return LogicalType(LogicalTypeId::DOUBLE);
		} else if (std::is_same<T, string_t>()) {
			return LogicalType(LogicalTypeId::VARCHAR);
		} else { // LCOV_EXCL_START
			throw std::runtime_error("Unrecognized type!");
		} // LCOV_EXCL_STOP
	}

	template <typename TA, typename TB, typename... Args>
	inline static void GetArgumentTypesRecursive(vector<LogicalType> &arguments) {
		arguments.push_back(GetArgumentType<TA>());
		GetArgumentTypesRecursive<TB, Args...>(arguments);
	}

	template <typename TA>
	inline static void GetArgumentTypesRecursive(vector<LogicalType> &arguments) {
		arguments.push_back(GetArgumentType<TA>());
	}

private:
	//-------------------------------- Argumented functions --------------------------------//

	template <typename TR, typename... Args>
	inline static scalar_function_t CreateUnaryFunction(const string &name, vector<LogicalType> args,
	                                                    LogicalType ret_type,
	                                                    TR (*udf_func)(Args...)) { // LCOV_EXCL_START
		throw std::runtime_error("Incorrect number of arguments for unary function");
	} // LCOV_EXCL_STOP

	template <typename TR, typename TA>
	inline static scalar_function_t CreateUnaryFunction(const string &name, vector<LogicalType> args,
	                                                    LogicalType ret_type, TR (*udf_func)(TA)) {
		if (args.size() != 1) { // LCOV_EXCL_START
			throw std::runtime_error("The number of LogicalType arguments (\"args\") should be 1!");
		}
		if (!TypesMatch<TA>(args[0])) {
			throw std::runtime_error("The first arguments don't match!");
		} // LCOV_EXCL_STOP

		scalar_function_t udf_function = [=](DataChunk &input, ExpressionState &state, Vector &result) -> void {
			UnaryExecutor::GenericExecute<TA, TR, UnaryUDFExecutor>(input.data[0], result, input.size(),
			                                                        (void *)udf_func);
		};
		return udf_function;
	}

	template <typename TR, typename... Args>
	inline static scalar_function_t CreateBinaryFunction(const string &name, vector<LogicalType> args,
	                                                     LogicalType ret_type,
	                                                     TR (*udf_func)(Args...)) { // LCOV_EXCL_START
		throw std::runtime_error("Incorrect number of arguments for binary function");
	} // LCOV_EXCL_STOP

	template <typename TR, typename TA, typename TB>
	inline static scalar_function_t CreateBinaryFunction(const string &name, vector<LogicalType> args,
	                                                     LogicalType ret_type, TR (*udf_func)(TA, TB)) {
		if (args.size() != 2) { // LCOV_EXCL_START
			throw std::runtime_error("The number of LogicalType arguments (\"args\") should be 2!");
		}
		if (!TypesMatch<TA>(args[0])) {
			throw std::runtime_error("The first arguments don't match!");
		}
		if (!TypesMatch<TB>(args[1])) {
			throw std::runtime_error("The second arguments don't match!");
		} // LCOV_EXCL_STOP

		scalar_function_t udf_function = [=](DataChunk &input, ExpressionState &state, Vector &result) {
			BinaryExecutor::Execute<TA, TB, TR>(input.data[0], input.data[1], result, input.size(), udf_func);
		};
		return udf_function;
	}

	template <typename TR, typename... Args>
	inline static scalar_function_t CreateTernaryFunction(const string &name, vector<LogicalType> args,
	                                                      LogicalType ret_type,
	                                                      TR (*udf_func)(Args...)) { // LCOV_EXCL_START
		throw std::runtime_error("Incorrect number of arguments for ternary function");
	} // LCOV_EXCL_STOP

	template <typename TR, typename TA, typename TB, typename TC>
	inline static scalar_function_t CreateTernaryFunction(const string &name, vector<LogicalType> args,
	                                                      LogicalType ret_type, TR (*udf_func)(TA, TB, TC)) {
		if (args.size() != 3) { // LCOV_EXCL_START
			throw std::runtime_error("The number of LogicalType arguments (\"args\") should be 3!");
		}
		if (!TypesMatch<TA>(args[0])) {
			throw std::runtime_error("The first arguments don't match!");
		}
		if (!TypesMatch<TB>(args[1])) {
			throw std::runtime_error("The second arguments don't match!");
		}
		if (!TypesMatch<TC>(args[2])) {
			throw std::runtime_error("The second arguments don't match!");
		} // LCOV_EXCL_STOP

		scalar_function_t udf_function = [=](DataChunk &input, ExpressionState &state, Vector &result) -> void {
			TernaryExecutor::Execute<TA, TB, TC, TR>(input.data[0], input.data[1], input.data[2], result, input.size(),
			                                         udf_func);
		};
		return udf_function;
	}

	template <typename T>
	inline static bool TypesMatch(const LogicalType &sql_type) {
		switch (sql_type.id()) {
		case LogicalTypeId::BOOLEAN:
			return std::is_same<T, bool>();
		case LogicalTypeId::TINYINT:
			return std::is_same<T, int8_t>();
		case LogicalTypeId::SMALLINT:
			return std::is_same<T, int16_t>();
		case LogicalTypeId::INTEGER:
			return std::is_same<T, int32_t>();
		case LogicalTypeId::BIGINT:
			return std::is_same<T, int64_t>();
		case LogicalTypeId::DATE:
			return std::is_same<T, date_t>();
		case LogicalTypeId::TIME:
		case LogicalTypeId::TIME_TZ:
			return std::is_same<T, dtime_t>();
		case LogicalTypeId::TIMESTAMP:
		case LogicalTypeId::TIMESTAMP_MS:
		case LogicalTypeId::TIMESTAMP_NS:
		case LogicalTypeId::TIMESTAMP_SEC:
		case LogicalTypeId::TIMESTAMP_TZ:
			return std::is_same<T, timestamp_t>();
		case LogicalTypeId::FLOAT:
			return std::is_same<T, float>();
		case LogicalTypeId::DOUBLE:
			return std::is_same<T, double>();
		case LogicalTypeId::VARCHAR:
		case LogicalTypeId::CHAR:
		case LogicalTypeId::BLOB:
			return std::is_same<T, string_t>();
		default: // LCOV_EXCL_START
			throw std::runtime_error("Type is not supported!");
		} // LCOV_EXCL_STOP
	}

private:
	//-------------------------------- Aggregate functions --------------------------------//
	template <typename UDF_OP, typename STATE, typename TR, typename TA>
	inline static AggregateFunction CreateUnaryAggregateFunction(const string &name) {
		LogicalType return_type = GetArgumentType<TR>();
		LogicalType input_type = GetArgumentType<TA>();
		return CreateUnaryAggregateFunction<UDF_OP, STATE, TR, TA>(name, return_type, input_type);
	}

	template <typename UDF_OP, typename STATE, typename TR, typename TA>
	inline static AggregateFunction CreateUnaryAggregateFunction(const string &name, LogicalType ret_type,
	                                                             LogicalType input_type) {
		AggregateFunction aggr_function =
		    AggregateFunction::UnaryAggregate<STATE, TR, TA, UDF_OP>(input_type, ret_type);
		aggr_function.name = name;
		return aggr_function;
	}

	template <typename UDF_OP, typename STATE, typename TR, typename TA, typename TB>
	inline static AggregateFunction CreateBinaryAggregateFunction(const string &name) {
		LogicalType return_type = GetArgumentType<TR>();
		LogicalType input_typeA = GetArgumentType<TA>();
		LogicalType input_typeB = GetArgumentType<TB>();
		return CreateBinaryAggregateFunction<UDF_OP, STATE, TR, TA, TB>(name, return_type, input_typeA, input_typeB);
	}

	template <typename UDF_OP, typename STATE, typename TR, typename TA, typename TB>
	inline static AggregateFunction CreateBinaryAggregateFunction(const string &name, LogicalType ret_type,
	                                                              LogicalType input_typeA, LogicalType input_typeB) {
		AggregateFunction aggr_function =
		    AggregateFunction::BinaryAggregate<STATE, TR, TA, TB, UDF_OP>(input_typeA, input_typeB, ret_type);
		aggr_function.name = name;
		return aggr_function;
	}
}; // end UDFWrapper

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/main/materialized_query_result.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/types/column/column_data_collection.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/pair.hpp
//
//
//===----------------------------------------------------------------------===//



#include <utility>

namespace duckdb {
using std::make_pair;
using std::pair;
} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/types/column/column_data_collection_iterators.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/types/column/column_data_scan_states.hpp
//
//
//===----------------------------------------------------------------------===//








namespace duckdb {

enum class ColumnDataAllocatorType : uint8_t {
	//! Use a buffer manager to allocate large chunks of memory that vectors then use
	BUFFER_MANAGER_ALLOCATOR,
	//! Use an in-memory allocator, allocating data for every chunk
	//! This causes the column data collection to allocate blocks that are not tied to a buffer manager
	IN_MEMORY_ALLOCATOR
};

enum class ColumnDataScanProperties : uint8_t {
	INVALID,
	//! Allow zero copy scans - this introduces a dependency on the resulting vector on the scan state of the column
	//! data collection, which means vectors might not be valid anymore after the next chunk is scanned.
	ALLOW_ZERO_COPY,
	//! Disallow zero-copy scans, always copying data into the target vector
	//! As a result, data scanned will be valid even after the column data collection is destroyed
	DISALLOW_ZERO_COPY
};

struct ChunkManagementState {
	unordered_map<idx_t, BufferHandle> handles;
	ColumnDataScanProperties properties = ColumnDataScanProperties::INVALID;
};

struct ColumnDataAppendState {
	ChunkManagementState current_chunk_state;
	vector<UnifiedVectorFormat> vector_data;
};

struct ColumnDataScanState {
	ChunkManagementState current_chunk_state;
	idx_t segment_index;
	idx_t chunk_index;
	idx_t current_row_index;
	idx_t next_row_index;
	ColumnDataScanProperties properties;
	vector<column_t> column_ids;
};

struct ColumnDataParallelScanState {
	ColumnDataScanState scan_state;
	mutex lock;
};

struct ColumnDataLocalScanState {
	ChunkManagementState current_chunk_state;
	idx_t current_segment_index = DConstants::INVALID_INDEX;
	idx_t current_row_index;
};

class ColumnDataRow {
public:
	ColumnDataRow(DataChunk &chunk, idx_t row_index, idx_t base_index);

	DataChunk &chunk;
	idx_t row_index;
	idx_t base_index;

public:
	Value GetValue(idx_t column_index) const;
	idx_t RowIndex() const;
};

} // namespace duckdb


namespace duckdb {
class ColumnDataCollection;

class ColumnDataChunkIterationHelper {
public:
	DUCKDB_API ColumnDataChunkIterationHelper(const ColumnDataCollection &collection, vector<column_t> column_ids);

private:
	const ColumnDataCollection &collection;
	vector<column_t> column_ids;

private:
	class ColumnDataChunkIterator;

	class ColumnDataChunkIterator {
	public:
		DUCKDB_API explicit ColumnDataChunkIterator(const ColumnDataCollection *collection_p,
		                                            vector<column_t> column_ids);

		const ColumnDataCollection *collection;
		ColumnDataScanState scan_state;
		shared_ptr<DataChunk> scan_chunk;
		idx_t row_index;

	public:
		DUCKDB_API void Next();

		DUCKDB_API ColumnDataChunkIterator &operator++();
		DUCKDB_API bool operator!=(const ColumnDataChunkIterator &other) const;
		DUCKDB_API DataChunk &operator*() const;
	};

public:
	ColumnDataChunkIterator begin() {
		return ColumnDataChunkIterator(&collection, column_ids);
	}
	ColumnDataChunkIterator end() {
		return ColumnDataChunkIterator(nullptr, vector<column_t>());
	}
};

class ColumnDataRowIterationHelper {
public:
	DUCKDB_API ColumnDataRowIterationHelper(const ColumnDataCollection &collection);

private:
	const ColumnDataCollection &collection;

private:
	class ColumnDataRowIterator;

	class ColumnDataRowIterator {
	public:
		DUCKDB_API explicit ColumnDataRowIterator(const ColumnDataCollection *collection_p);

		const ColumnDataCollection *collection;
		ColumnDataScanState scan_state;
		shared_ptr<DataChunk> scan_chunk;
		ColumnDataRow current_row;

	public:
		void Next();

		DUCKDB_API ColumnDataRowIterator &operator++();
		DUCKDB_API bool operator!=(const ColumnDataRowIterator &other) const;
		DUCKDB_API const ColumnDataRow &operator*() const;
	};

public:
	DUCKDB_API ColumnDataRowIterator begin();
	DUCKDB_API ColumnDataRowIterator end();
};

} // namespace duckdb


namespace duckdb {
class BufferManager;
class BlockHandle;
class ClientContext;
struct ColumnDataCopyFunction;
class ColumnDataAllocator;
class ColumnDataCollection;
class ColumnDataCollectionSegment;
class ColumnDataRowCollection;

//! The ColumnDataCollection represents a set of (buffer-managed) data stored in columnar format
//! It is efficient to read and scan
class ColumnDataCollection {
public:
	//! Constructs an in-memory column data collection from an allocator
	DUCKDB_API ColumnDataCollection(Allocator &allocator, vector<LogicalType> types);
	//! Constructs an empty (but valid) in-memory column data collection from an allocator
	DUCKDB_API ColumnDataCollection(Allocator &allocator);
	//! Constructs a buffer-managed column data collection
	DUCKDB_API ColumnDataCollection(BufferManager &buffer_manager, vector<LogicalType> types);
	//! Constructs either an in-memory or a buffer-managed column data collection
	DUCKDB_API ColumnDataCollection(ClientContext &context, vector<LogicalType> types,
	                                ColumnDataAllocatorType type = ColumnDataAllocatorType::BUFFER_MANAGER_ALLOCATOR);
	//! Creates a column data collection that inherits the blocks to write to. This allows blocks to be shared
	//! between multiple column data collections and prevents wasting space.
	//! Note that after one CDC inherits blocks from another, the other
	//! cannot be written to anymore (i.e. we take ownership of the half-written blocks).
	DUCKDB_API ColumnDataCollection(ColumnDataCollection &parent);
	DUCKDB_API ColumnDataCollection(shared_ptr<ColumnDataAllocator> allocator, vector<LogicalType> types);
	DUCKDB_API ~ColumnDataCollection();

public:
	//! The types of columns in the ColumnDataCollection
	vector<LogicalType> &Types() {
		return types;
	}
	const vector<LogicalType> &Types() const {
		return types;
	}

	//! The amount of rows in the ColumnDataCollection
	const idx_t &Count() const {
		return count;
	}

	//! The amount of columns in the ColumnDataCollection
	idx_t ColumnCount() const {
		return types.size();
	}

	//! Get the allocator
	DUCKDB_API Allocator &GetAllocator() const;

	//! Initializes an Append state - useful for optimizing many appends made to the same column data collection
	DUCKDB_API void InitializeAppend(ColumnDataAppendState &state);
	//! Append a DataChunk to this ColumnDataCollection using the specified append state
	DUCKDB_API void Append(ColumnDataAppendState &state, DataChunk &new_chunk);

	//! Initializes a chunk with the correct types that can be used to call Scan
	DUCKDB_API void InitializeScanChunk(DataChunk &chunk) const;
	//! Initializes a chunk with the correct types for a given scan state
	DUCKDB_API void InitializeScanChunk(ColumnDataScanState &state, DataChunk &chunk) const;
	//! Initializes a Scan state for scanning all columns
	DUCKDB_API void
	InitializeScan(ColumnDataScanState &state,
	               ColumnDataScanProperties properties = ColumnDataScanProperties::ALLOW_ZERO_COPY) const;
	//! Initializes a Scan state for scanning a subset of the columns
	DUCKDB_API void
	InitializeScan(ColumnDataScanState &state, vector<column_t> column_ids,
	               ColumnDataScanProperties properties = ColumnDataScanProperties::ALLOW_ZERO_COPY) const;
	//! Initialize a parallel scan over the column data collection over all columns
	DUCKDB_API void
	InitializeScan(ColumnDataParallelScanState &state,
	               ColumnDataScanProperties properties = ColumnDataScanProperties::ALLOW_ZERO_COPY) const;
	//! Initialize a parallel scan over the column data collection over a subset of the columns
	DUCKDB_API void
	InitializeScan(ColumnDataParallelScanState &state, vector<column_t> column_ids,
	               ColumnDataScanProperties properties = ColumnDataScanProperties::ALLOW_ZERO_COPY) const;
	//! Scans a DataChunk from the ColumnDataCollection
	DUCKDB_API bool Scan(ColumnDataScanState &state, DataChunk &result) const;
	//! Scans a DataChunk from the ColumnDataCollection
	DUCKDB_API bool Scan(ColumnDataParallelScanState &state, ColumnDataLocalScanState &lstate, DataChunk &result) const;

	//! Append a DataChunk directly to this ColumnDataCollection - calls InitializeAppend and Append internally
	DUCKDB_API void Append(DataChunk &new_chunk);

	//! Appends the other ColumnDataCollection to this, destroying the other data collection
	DUCKDB_API void Combine(ColumnDataCollection &other);

	DUCKDB_API void Verify();

	DUCKDB_API string ToString() const;
	DUCKDB_API void Print() const;

	DUCKDB_API void Reset();

	//! Returns the number of data chunks present in the ColumnDataCollection
	DUCKDB_API idx_t ChunkCount() const;
	//! Fetch an individual chunk from the ColumnDataCollection
	DUCKDB_API void FetchChunk(idx_t chunk_idx, DataChunk &result) const;

	//! Constructs a class that can be iterated over to fetch individual chunks
	//! Iterating over this is syntactic sugar over just calling Scan
	DUCKDB_API ColumnDataChunkIterationHelper Chunks() const;
	//! Constructs a class that can be iterated over to fetch individual chunks
	//! Only the column indexes specified in the column_ids list are scanned
	DUCKDB_API ColumnDataChunkIterationHelper Chunks(vector<column_t> column_ids) const;

	//! Constructs a class that can be iterated over to fetch individual rows
	//! Note that row iteration is slow, and the `.Chunks()` method should be used instead
	DUCKDB_API ColumnDataRowIterationHelper Rows() const;

	//! Returns a materialized set of all of the rows in the column data collection
	//! Note that usage of this is slow - avoid using this unless the amount of rows is small, or if you do not care
	//! about performance
	DUCKDB_API ColumnDataRowCollection GetRows() const;

	//! Compare two column data collections to another. If they are equal according to result equality rules,
	//! return true. That means null values are equal, and approx equality is used for floating point values.
	//! If they are not equal, return false and fill in the error message.
	static bool ResultEquals(const ColumnDataCollection &left, const ColumnDataCollection &right, string &error_message,
	                         bool ordered = false);

	//! Obtains the next scan index to scan from
	bool NextScanIndex(ColumnDataScanState &state, idx_t &chunk_index, idx_t &segment_index, idx_t &row_index) const;
	//! Scans at the indices (obtained from NextScanIndex)
	void ScanAtIndex(ColumnDataParallelScanState &state, ColumnDataLocalScanState &lstate, DataChunk &result,
	                 idx_t chunk_index, idx_t segment_index, idx_t row_index) const;

	//! Initialize the column data collection
	void Initialize(vector<LogicalType> types);

	//! Get a vector of references to every chunk (segment, index in segment), and optionally sort by block id
	const vector<unique_ptr<ColumnDataCollectionSegment>> &GetSegments() const;

private:
	//! Creates a new segment within the ColumnDataCollection
	void CreateSegment();

	static ColumnDataCopyFunction GetCopyFunction(const LogicalType &type);

private:
	//! The Column Data Allocator
	buffer_ptr<ColumnDataAllocator> allocator;
	//! The types of the stored entries
	vector<LogicalType> types;
	//! The number of entries stored in the column data collection
	idx_t count;
	//! The data segments of the column data collection
	vector<unique_ptr<ColumnDataCollectionSegment>> segments;
	//! The set of copy functions
	vector<ColumnDataCopyFunction> copy_functions;
	//! When the column data collection is marked as finished - new tuples can no longer be appended to it
	bool finished_append;
};

//! The ColumnDataRowCollection represents a set of materialized rows, as obtained from the ColumnDataCollection
class ColumnDataRowCollection {
public:
	DUCKDB_API ColumnDataRowCollection(const ColumnDataCollection &collection);

public:
	DUCKDB_API Value GetValue(idx_t column, idx_t index) const;

public:
	// container API
	bool empty() const {
		return rows.empty();
	}
	idx_t size() const {
		return rows.size();
	}

	DUCKDB_API ColumnDataRow &operator[](idx_t i);
	DUCKDB_API const ColumnDataRow &operator[](idx_t i) const;

	vector<ColumnDataRow>::iterator begin() {
		return rows.begin();
	}
	vector<ColumnDataRow>::iterator end() {
		return rows.end();
	}
	vector<ColumnDataRow>::const_iterator cbegin() const {
		return rows.cbegin();
	}
	vector<ColumnDataRow>::const_iterator cend() const {
		return rows.cend();
	}
	vector<ColumnDataRow>::const_iterator begin() const {
		return rows.begin();
	}
	vector<ColumnDataRow>::const_iterator end() const {
		return rows.end();
	}

private:
	vector<ColumnDataRow> rows;
	vector<unique_ptr<DataChunk>> chunks;
	ColumnDataScanState scan_state;
};

} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/main/query_result.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/statement_type.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {

//===--------------------------------------------------------------------===//
// Statement Types
//===--------------------------------------------------------------------===//
enum class StatementType : uint8_t {
	INVALID_STATEMENT,      // invalid statement type
	SELECT_STATEMENT,       // select statement type
	INSERT_STATEMENT,       // insert statement type
	UPDATE_STATEMENT,       // update statement type
	CREATE_STATEMENT,       // create statement type
	DELETE_STATEMENT,       // delete statement type
	PREPARE_STATEMENT,      // prepare statement type
	EXECUTE_STATEMENT,      // execute statement type
	ALTER_STATEMENT,        // alter statement type
	TRANSACTION_STATEMENT,  // transaction statement type,
	COPY_STATEMENT,         // copy type
	ANALYZE_STATEMENT,      // analyze type
	VARIABLE_SET_STATEMENT, // variable set statement type
	CREATE_FUNC_STATEMENT,  // create func statement type
	EXPLAIN_STATEMENT,      // explain statement type
	DROP_STATEMENT,         // DROP statement type
	EXPORT_STATEMENT,       // EXPORT statement type
	PRAGMA_STATEMENT,       // PRAGMA statement type
	SHOW_STATEMENT,         // SHOW statement type
	VACUUM_STATEMENT,       // VACUUM statement type
	CALL_STATEMENT,         // CALL statement type
	SET_STATEMENT,          // SET statement type
	LOAD_STATEMENT,         // LOAD statement type
	RELATION_STATEMENT,
	EXTENSION_STATEMENT,
	LOGICAL_PLAN_STATEMENT,
	ATTACH_STATEMENT,
	DETACH_STATEMENT,
	MULTI_STATEMENT

};

DUCKDB_API string StatementTypeToString(StatementType type);

enum class StatementReturnType : uint8_t {
	QUERY_RESULT, // the statement returns a query result (e.g. for display to the user)
	CHANGED_ROWS, // the statement returns a single row containing the number of changed rows (e.g. an insert stmt)
	NOTHING       // the statement returns nothing
};

string StatementReturnTypeToString(StatementReturnType type);

//! A struct containing various properties of a SQL statement
struct StatementProperties {
	StatementProperties()
	    : requires_valid_transaction(true), allow_stream_result(false), bound_all_parameters(true),
	      return_type(StatementReturnType::QUERY_RESULT), parameter_count(0) {
	}

	//! The set of databases this statement will modify
	unordered_set<string> modified_databases;
	//! Whether or not the statement requires a valid transaction. Almost all statements require this, with the
	//! exception of
	bool requires_valid_transaction;
	//! Whether or not the result can be streamed to the client
	bool allow_stream_result;
	//! Whether or not all parameters have successfully had their types determined
	bool bound_all_parameters;
	//! What type of data the statement returns
	StatementReturnType return_type;
	//! The number of prepared statement parameters
	idx_t parameter_count;

	bool IsReadOnly() {
		return modified_databases.empty();
	}
};

} // namespace duckdb




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/arrow/arrow_options.hpp
//
//
//===----------------------------------------------------------------------===//



namespace duckdb {

enum ArrowOffsetSize { REGULAR, LARGE };

struct ArrowOptions {
	explicit ArrowOptions(ArrowOffsetSize offset_size_p) : offset_size(offset_size_p) {
	}
	ArrowOptions(ArrowOffsetSize offset_size_p, string timezone_p) : offset_size(offset_size_p), time_zone(timezone_p) {
	}
	ArrowOptions() {
	}
	ArrowOffsetSize offset_size = ArrowOffsetSize::REGULAR;
	string time_zone = "UTC";
};
} // namespace duckdb


namespace duckdb {
struct BoxRendererConfig;

enum class QueryResultType : uint8_t { MATERIALIZED_RESULT, STREAM_RESULT, PENDING_RESULT };

//! A set of properties from the client context that can be used to interpret the query result
struct ClientProperties {
	ClientProperties(string time_zone_p, ArrowOffsetSize arrow_offset_size_p)
	    : time_zone(std::move(time_zone_p)), arrow_offset_size(arrow_offset_size_p) {
	}
	string time_zone;
	ArrowOffsetSize arrow_offset_size;
};

class BaseQueryResult {
public:
	//! Creates a successful query result with the specified names and types
	DUCKDB_API BaseQueryResult(QueryResultType type, StatementType statement_type, StatementProperties properties,
	                           vector<LogicalType> types, vector<string> names);
	//! Creates an unsuccessful query result with error condition
	DUCKDB_API BaseQueryResult(QueryResultType type, PreservedError error);
	DUCKDB_API virtual ~BaseQueryResult();

	//! The type of the result (MATERIALIZED or STREAMING)
	QueryResultType type;
	//! The type of the statement that created this result
	StatementType statement_type;
	//! Properties of the statement
	StatementProperties properties;
	//! The SQL types of the result
	vector<LogicalType> types;
	//! The names of the result
	vector<string> names;

public:
	DUCKDB_API void ThrowError(const string &prepended_message = "") const;
	DUCKDB_API void SetError(PreservedError error);
	DUCKDB_API bool HasError() const;
	DUCKDB_API const ExceptionType &GetErrorType() const;
	DUCKDB_API const std::string &GetError();
	DUCKDB_API PreservedError &GetErrorObject();
	DUCKDB_API idx_t ColumnCount();

protected:
	//! Whether or not execution was successful
	bool success;
	//! The error (in case execution was not successful)
	PreservedError error;
};
struct CurrentChunk {
	//! The current data chunk
	unique_ptr<DataChunk> data_chunk;
	//! The current position in the data chunk
	idx_t position;
	//! If we have a current chunk we must scan for result production
	bool Valid();
	//! The remaining size of the current chunk
	idx_t RemainingSize();
};
//! The QueryResult object holds the result of a query. It can either be a MaterializedQueryResult, in which case the
//! result contains the entire result set, or a StreamQueryResult in which case the Fetch method can be called to
//! incrementally fetch data from the database.
class QueryResult : public BaseQueryResult {
public:
	//! Creates a successful query result with the specified names and types
	DUCKDB_API QueryResult(QueryResultType type, StatementType statement_type, StatementProperties properties,
	                       vector<LogicalType> types, vector<string> names, ClientProperties client_properties);
	//! Creates an unsuccessful query result with error condition
	DUCKDB_API QueryResult(QueryResultType type, PreservedError error);
	DUCKDB_API virtual ~QueryResult() override;

	//! Properties from the client context
	ClientProperties client_properties;
	//! The next result (if any)
	unique_ptr<QueryResult> next;
	//! In case we are converting the result from Native DuckDB to a different library (e.g., Arrow, Polars)
	//! We might be producing chunks of a pre-determined size.
	//! To comply, we use the following variable to store the current chunk, and it's position.
	CurrentChunk current_chunk;

public:
	template <class TARGET>
	TARGET &Cast() {
		if (type != TARGET::TYPE) {
			throw InternalException("Failed to cast query result to type - query result type mismatch");
		}
		return reinterpret_cast<TARGET &>(*this);
	}

	template <class TARGET>
	const TARGET &Cast() const {
		if (type != TARGET::TYPE) {
			throw InternalException("Failed to cast query result to type - query result type mismatch");
		}
		return reinterpret_cast<const TARGET &>(*this);
	}

public:
	//! Returns the name of the column for the given index
	DUCKDB_API const string &ColumnName(idx_t index) const;
	//! Fetches a DataChunk of normalized (flat) vectors from the query result.
	//! Returns nullptr if there are no more results to fetch.
	DUCKDB_API virtual unique_ptr<DataChunk> Fetch();
	//! Fetches a DataChunk from the query result. The vectors are not normalized and hence any vector types can be
	//! returned.
	DUCKDB_API virtual unique_ptr<DataChunk> FetchRaw() = 0;
	//! Converts the QueryResult to a string
	DUCKDB_API virtual string ToString() = 0;
	//! Converts the QueryResult to a box-rendered string
	DUCKDB_API virtual string ToBox(ClientContext &context, const BoxRendererConfig &config);
	//! Prints the QueryResult to the console
	DUCKDB_API void Print();
	//! Returns true if the two results are identical; false otherwise. Note that this method is destructive; it calls
	//! Fetch() until both results are exhausted. The data in the results will be lost.
	DUCKDB_API bool Equals(QueryResult &other);

	bool TryFetch(unique_ptr<DataChunk> &result, PreservedError &error) {
		try {
			result = Fetch();
			return success;
		} catch (const Exception &ex) {
			error = PreservedError(ex);
			return false;
		} catch (std::exception &ex) {
			error = PreservedError(ex);
			return false;
		} catch (...) {
			error = PreservedError("Unknown error in Fetch");
			return false;
		}
	}

	static ArrowOptions GetArrowOptions(QueryResult &query_result);
	static string GetConfigTimezone(QueryResult &query_result);

private:
	class QueryResultIterator;
	class QueryResultRow {
	public:
		explicit QueryResultRow(QueryResultIterator &iterator_p, idx_t row_idx) : iterator(iterator_p), row(0) {
		}

		QueryResultIterator &iterator;
		idx_t row;

		template <class T>
		T GetValue(idx_t col_idx) const {
			return iterator.chunk->GetValue(col_idx, row).GetValue<T>();
		}
	};
	//! The row-based query result iterator. Invoking the
	class QueryResultIterator {
	public:
		explicit QueryResultIterator(optional_ptr<QueryResult> result_p)
		    : current_row(*this, 0), result(result_p), base_row(0) {
			if (result) {
				chunk = shared_ptr<DataChunk>(result->Fetch().release());
				if (!chunk) {
					result = nullptr;
				}
			}
		}

		QueryResultRow current_row;
		shared_ptr<DataChunk> chunk;
		optional_ptr<QueryResult> result;
		idx_t base_row;

	public:
		void Next() {
			if (!chunk) {
				return;
			}
			current_row.row++;
			if (current_row.row >= chunk->size()) {
				base_row += chunk->size();
				chunk = shared_ptr<DataChunk>(result->Fetch().release());
				current_row.row = 0;
				if (!chunk || chunk->size() == 0) {
					// exhausted all rows
					base_row = 0;
					result = nullptr;
					chunk.reset();
				}
			}
		}

		QueryResultIterator &operator++() {
			Next();
			return *this;
		}
		bool operator!=(const QueryResultIterator &other) const {
			return result != other.result || base_row != other.base_row || current_row.row != other.current_row.row;
		}
		const QueryResultRow &operator*() const {
			return current_row;
		}
	};

public:
	QueryResultIterator begin() {
		return QueryResultIterator(this);
	}
	QueryResultIterator end() {
		return QueryResultIterator(nullptr);
	}

protected:
	DUCKDB_API string HeaderToString();

private:
	QueryResult(const QueryResult &) = delete;
};

} // namespace duckdb


namespace duckdb {

class ClientContext;

class MaterializedQueryResult : public QueryResult {
public:
	static constexpr const QueryResultType TYPE = QueryResultType::MATERIALIZED_RESULT;

public:
	friend class ClientContext;
	//! Creates a successful query result with the specified names and types
	DUCKDB_API MaterializedQueryResult(StatementType statement_type, StatementProperties properties,
	                                   vector<string> names, unique_ptr<ColumnDataCollection> collection,
	                                   ClientProperties client_properties);
	//! Creates an unsuccessful query result with error condition
	DUCKDB_API explicit MaterializedQueryResult(PreservedError error);

public:
	//! Fetches a DataChunk from the query result.
	//! This will consume the result (i.e. the result can only be scanned once with this function)
	DUCKDB_API unique_ptr<DataChunk> Fetch() override;
	DUCKDB_API unique_ptr<DataChunk> FetchRaw() override;
	//! Converts the QueryResult to a string
	DUCKDB_API string ToString() override;
	DUCKDB_API string ToBox(ClientContext &context, const BoxRendererConfig &config) override;

	//! Gets the (index) value of the (column index) column.
	//! Note: this is very slow. Scanning over the underlying collection is much faster.
	DUCKDB_API Value GetValue(idx_t column, idx_t index);

	template <class T>
	T GetValue(idx_t column, idx_t index) {
		auto value = GetValue(column, index);
		return (T)value.GetValue<int64_t>();
	}

	DUCKDB_API idx_t RowCount() const;

	//! Returns a reference to the underlying column data collection
	ColumnDataCollection &Collection();

private:
	unique_ptr<ColumnDataCollection> collection;
	//! Row collection, only created if GetValue is called
	unique_ptr<ColumnDataRowCollection> row_collection;
	//! Scan state for Fetch calls
	ColumnDataScanState scan_state;
	bool scan_initialized;
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/main/pending_query_result.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/pending_execution_result.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

enum class PendingExecutionResult : uint8_t { RESULT_READY, RESULT_NOT_READY, EXECUTION_ERROR };

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/execution/executor.hpp
//
//
//===----------------------------------------------------------------------===//








//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parallel/pipeline.hpp
//
//
//===----------------------------------------------------------------------===//





//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/set.hpp
//
//
//===----------------------------------------------------------------------===//



#include <set>

namespace duckdb {
using std::multiset;
using std::set;
} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/execution/physical_operator.hpp
//
//
//===----------------------------------------------------------------------===//





//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/operator_result_type.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

//! The OperatorResultType is used to indicate how data should flow around a regular (i.e. non-sink and non-source)
//! physical operator
//! There are four possible results:
//! NEED_MORE_INPUT means the operator is done with the current input and can consume more input if available
//! If there is more input the operator will be called with more input, otherwise the operator will not be called again.
//! HAVE_MORE_OUTPUT means the operator is not finished yet with the current input.
//! The operator will be called again with the same input.
//! FINISHED means the operator has finished the entire pipeline and no more processing is necessary.
//! The operator will not be called again, and neither will any other operators in this pipeline.
//! BLOCKED means the operator does not want to be called right now. e.g. because its currently doing async I/O. The
//! operator has set the interrupt state and the caller is expected to handle it. Note that intermediate operators
//! should currently not emit this state.
enum class OperatorResultType : uint8_t { NEED_MORE_INPUT, HAVE_MORE_OUTPUT, FINISHED, BLOCKED };

//! OperatorFinalizeResultType is used to indicate whether operators have finished flushing their cached results.
//! FINISHED means the operator has flushed all cached data.
//! HAVE_MORE_OUTPUT means the operator contains more results.
enum class OperatorFinalizeResultType : uint8_t { HAVE_MORE_OUTPUT, FINISHED };

//! SourceResultType is used to indicate the result of data being pulled out of a source.
//! There are three possible results:
//! HAVE_MORE_OUTPUT means the source has more output, this flag should only be set when data is returned, empty results
//! should only occur for the FINISHED and BLOCKED flags
//! FINISHED means the source is exhausted
//! BLOCKED means the source is currently blocked, e.g. by some async I/O
enum class SourceResultType : uint8_t { HAVE_MORE_OUTPUT, FINISHED, BLOCKED };

//! The SinkResultType is used to indicate the result of data flowing into a sink
//! There are three possible results:
//! NEED_MORE_INPUT means the sink needs more input
//! FINISHED means the sink is finished executing, and more input will not change the result any further
//! BLOCKED means the sink is currently blocked, e.g. by some async I/O.
enum class SinkResultType : uint8_t { NEED_MORE_INPUT, FINISHED, BLOCKED };

//! The SinkFinalizeType is used to indicate the result of a Finalize call on a sink
//! There are two possible results:
//! READY means the sink is ready for further processing
//! NO_OUTPUT_POSSIBLE means the sink will never provide output, and any pipelines involving the sink can be skipped
enum class SinkFinalizeType : uint8_t { READY, NO_OUTPUT_POSSIBLE };

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/physical_operator_type.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

//===--------------------------------------------------------------------===//
// Physical Operator Types
//===--------------------------------------------------------------------===//
enum class PhysicalOperatorType : uint8_t {
	INVALID,
	ORDER_BY,
	LIMIT,
	STREAMING_LIMIT,
	LIMIT_PERCENT,
	TOP_N,
	WINDOW,
	UNNEST,
	UNGROUPED_AGGREGATE,
	HASH_GROUP_BY,
	PERFECT_HASH_GROUP_BY,
	FILTER,
	PROJECTION,
	COPY_TO_FILE,
	BATCH_COPY_TO_FILE,
	FIXED_BATCH_COPY_TO_FILE,
	RESERVOIR_SAMPLE,
	STREAMING_SAMPLE,
	STREAMING_WINDOW,
	PIVOT,

	// -----------------------------
	// Scans
	// -----------------------------
	TABLE_SCAN,
	DUMMY_SCAN,
	COLUMN_DATA_SCAN,
	CHUNK_SCAN,
	RECURSIVE_CTE_SCAN,
	DELIM_SCAN,
	EXPRESSION_SCAN,
	POSITIONAL_SCAN,
	// -----------------------------
	// Joins
	// -----------------------------
	BLOCKWISE_NL_JOIN,
	NESTED_LOOP_JOIN,
	HASH_JOIN,
	CROSS_PRODUCT,
	PIECEWISE_MERGE_JOIN,
	IE_JOIN,
	DELIM_JOIN,
	INDEX_JOIN,
	POSITIONAL_JOIN,
	ASOF_JOIN,
	// -----------------------------
	// SetOps
	// -----------------------------
	UNION,
	RECURSIVE_CTE,

	// -----------------------------
	// Updates
	// -----------------------------
	INSERT,
	BATCH_INSERT,
	DELETE_OPERATOR,
	UPDATE,

	// -----------------------------
	// Schema
	// -----------------------------
	CREATE_TABLE,
	CREATE_TABLE_AS,
	BATCH_CREATE_TABLE_AS,
	CREATE_INDEX,
	ALTER,
	CREATE_SEQUENCE,
	CREATE_VIEW,
	CREATE_SCHEMA,
	CREATE_MACRO,
	DROP,
	PRAGMA,
	TRANSACTION,
	CREATE_TYPE,
	ATTACH,
	DETACH,

	// -----------------------------
	// Helpers
	// -----------------------------
	EXPLAIN,
	EXPLAIN_ANALYZE,
	EMPTY_RESULT,
	EXECUTE,
	PREPARE,
	VACUUM,
	EXPORT,
	SET,
	LOAD,
	INOUT_FUNCTION,
	RESULT_COLLECTOR,
	RESET,
	EXTENSION
};

string PhysicalOperatorToString(PhysicalOperatorType type);

} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/execution/execution_context.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {
class ClientContext;
class ThreadContext;
class Pipeline;

class ExecutionContext {
public:
	ExecutionContext(ClientContext &client_p, ThreadContext &thread_p, optional_ptr<Pipeline> pipeline_p)
	    : client(client_p), thread(thread_p), pipeline(pipeline_p) {
	}

	//! The client-global context; caution needs to be taken when used in parallel situations
	ClientContext &client;
	//! The thread-local context for this execution
	ThreadContext &thread;
	//! Reference to the pipeline for this execution, can be used for example by operators determine caching strategy
	optional_ptr<Pipeline> pipeline;
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/optimizer/join_order/join_node.hpp
//
//
//===----------------------------------------------------------------------===//





//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/optimizer/join_order/estimated_properties.hpp
//
//
//===----------------------------------------------------------------------===//





//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/optimizer/join_order/join_relation.hpp
//
//
//===----------------------------------------------------------------------===//








namespace duckdb {
class LogicalOperator;

//! Represents a single relation and any metadata accompanying that relation
struct SingleJoinRelation {
	LogicalOperator &op;
	optional_ptr<LogicalOperator> parent;

	SingleJoinRelation(LogicalOperator &op, optional_ptr<LogicalOperator> parent) : op(op), parent(parent) {
	}
};

//! Set of relations, used in the join graph.
struct JoinRelationSet {
	JoinRelationSet(unsafe_unique_array<idx_t> relations, idx_t count) : relations(std::move(relations)), count(count) {
	}

	string ToString() const;

	unsafe_unique_array<idx_t> relations;
	idx_t count;

	static bool IsSubset(JoinRelationSet &super, JoinRelationSet &sub);
};

//! The JoinRelationTree is a structure holding all the created JoinRelationSet objects and allowing fast lookup on to
//! them
class JoinRelationSetManager {
public:
	//! Contains a node with a JoinRelationSet and child relations
	// FIXME: this structure is inefficient, could use a bitmap for lookup instead (todo: profile)
	struct JoinRelationTreeNode {
		unique_ptr<JoinRelationSet> relation;
		unordered_map<idx_t, unique_ptr<JoinRelationTreeNode>> children;
	};

public:
	//! Create or get a JoinRelationSet from a single node with the given index
	JoinRelationSet &GetJoinRelation(idx_t index);
	//! Create or get a JoinRelationSet from a set of relation bindings
	JoinRelationSet &GetJoinRelation(unordered_set<idx_t> &bindings);
	//! Create or get a JoinRelationSet from a (sorted, duplicate-free!) list of relations
	JoinRelationSet &GetJoinRelation(unsafe_unique_array<idx_t> relations, idx_t count);
	//! Union two sets of relations together and create a new relation set
	JoinRelationSet &Union(JoinRelationSet &left, JoinRelationSet &right);
	// //! Create the set difference of left \ right (i.e. all elements in left that are not in right)
	// JoinRelationSet *Difference(JoinRelationSet *left, JoinRelationSet *right);

private:
	JoinRelationTreeNode root;
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/expression_map.hpp
//
//
//===----------------------------------------------------------------------===//









namespace duckdb {
class Expression;

template <class T>
struct ExpressionHashFunction {
	uint64_t operator()(const reference<T> &expr) const {
		return (uint64_t)expr.get().Hash();
	}
};

template <class T>
struct ExpressionEquality {
	bool operator()(const reference<T> &a, const reference<T> &b) const {
		return a.get().Equals(b.get());
	}
};

template <typename T>
using expression_map_t =
    unordered_map<reference<Expression>, T, ExpressionHashFunction<Expression>, ExpressionEquality<Expression>>;

using expression_set_t =
    unordered_set<reference<Expression>, ExpressionHashFunction<Expression>, ExpressionEquality<Expression>>;

template <typename T>
using parsed_expression_map_t = unordered_map<reference<ParsedExpression>, T, ExpressionHashFunction<ParsedExpression>,
                                              ExpressionEquality<ParsedExpression>>;

using parsed_expression_set_t = unordered_set<reference<ParsedExpression>, ExpressionHashFunction<ParsedExpression>,
                                              ExpressionEquality<ParsedExpression>>;

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/planner/logical_operator_visitor.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/planner/bound_tokens.hpp
//
//
//===----------------------------------------------------------------------===//



namespace duckdb {

//===--------------------------------------------------------------------===//
// Query Node
//===--------------------------------------------------------------------===//
class BoundQueryNode;
class BoundSelectNode;
class BoundSetOperationNode;
class BoundRecursiveCTENode;

//===--------------------------------------------------------------------===//
// Expressions
//===--------------------------------------------------------------------===//
class Expression;

class BoundAggregateExpression;
class BoundBetweenExpression;
class BoundCaseExpression;
class BoundCastExpression;
class BoundColumnRefExpression;
class BoundComparisonExpression;
class BoundConjunctionExpression;
class BoundConstantExpression;
class BoundDefaultExpression;
class BoundFunctionExpression;
class BoundOperatorExpression;
class BoundParameterExpression;
class BoundReferenceExpression;
class BoundSubqueryExpression;
class BoundUnnestExpression;
class BoundWindowExpression;

//===--------------------------------------------------------------------===//
// TableRefs
//===--------------------------------------------------------------------===//
class BoundTableRef;

class BoundBaseTableRef;
class BoundJoinRef;
class BoundSubqueryRef;
class BoundTableFunction;
class BoundEmptyTableRef;
class BoundExpressionListRef;
class BoundCTERef;
class BoundPivotRef;

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/planner/logical_tokens.hpp
//
//
//===----------------------------------------------------------------------===//



namespace duckdb {

class LogicalOperator;

class LogicalAggregate;
class LogicalAnyJoin;
class LogicalAsOfJoin;
class LogicalColumnDataGet;
class LogicalComparisonJoin;
class LogicalCopyToFile;
class LogicalCreate;
class LogicalCreateTable;
class LogicalCreateIndex;
class LogicalCreateTable;
class LogicalCrossProduct;
class LogicalCTERef;
class LogicalDelete;
class LogicalDelimGet;
class LogicalDelimJoin;
class LogicalDistinct;
class LogicalDummyScan;
class LogicalEmptyResult;
class LogicalExecute;
class LogicalExplain;
class LogicalExport;
class LogicalExpressionGet;
class LogicalFilter;
class LogicalGet;
class LogicalInsert;
class LogicalJoin;
class LogicalLimit;
class LogicalOrder;
class LogicalPivot;
class LogicalPositionalJoin;
class LogicalPragma;
class LogicalPrepare;
class LogicalProjection;
class LogicalRecursiveCTE;
class LogicalSetOperation;
class LogicalSample;
class LogicalShow;
class LogicalSimple;
class LogicalSet;
class LogicalReset;
class LogicalTopN;
class LogicalUnnest;
class LogicalUpdate;
class LogicalWindow;

} // namespace duckdb


#include <functional>

namespace duckdb {
//! The LogicalOperatorVisitor is an abstract base class that implements the
//! Visitor pattern on LogicalOperator.
class LogicalOperatorVisitor {
public:
	virtual ~LogicalOperatorVisitor() {};

	virtual void VisitOperator(LogicalOperator &op);
	virtual void VisitExpression(unique_ptr<Expression> *expression);

	static void EnumerateExpressions(LogicalOperator &op,
	                                 const std::function<void(unique_ptr<Expression> *child)> &callback);

protected:
	//! Automatically calls the Visit method for LogicalOperator children of the current operator. Can be overloaded to
	//! change this behavior.
	void VisitOperatorChildren(LogicalOperator &op);
	//! Automatically calls the Visit method for Expression children of the current operator. Can be overloaded to
	//! change this behavior.
	void VisitOperatorExpressions(LogicalOperator &op);

	// The VisitExpressionChildren method is called at the end of every call to VisitExpression to recursively visit all
	// expressions in an expression tree. It can be overloaded to prevent automatically visiting the entire tree.
	virtual void VisitExpressionChildren(Expression &expression);

	virtual unique_ptr<Expression> VisitReplace(BoundAggregateExpression &expr, unique_ptr<Expression> *expr_ptr);
	virtual unique_ptr<Expression> VisitReplace(BoundBetweenExpression &expr, unique_ptr<Expression> *expr_ptr);
	virtual unique_ptr<Expression> VisitReplace(BoundCaseExpression &expr, unique_ptr<Expression> *expr_ptr);
	virtual unique_ptr<Expression> VisitReplace(BoundCastExpression &expr, unique_ptr<Expression> *expr_ptr);
	virtual unique_ptr<Expression> VisitReplace(BoundColumnRefExpression &expr, unique_ptr<Expression> *expr_ptr);
	virtual unique_ptr<Expression> VisitReplace(BoundComparisonExpression &expr, unique_ptr<Expression> *expr_ptr);
	virtual unique_ptr<Expression> VisitReplace(BoundConjunctionExpression &expr, unique_ptr<Expression> *expr_ptr);
	virtual unique_ptr<Expression> VisitReplace(BoundConstantExpression &expr, unique_ptr<Expression> *expr_ptr);
	virtual unique_ptr<Expression> VisitReplace(BoundDefaultExpression &expr, unique_ptr<Expression> *expr_ptr);
	virtual unique_ptr<Expression> VisitReplace(BoundFunctionExpression &expr, unique_ptr<Expression> *expr_ptr);
	virtual unique_ptr<Expression> VisitReplace(BoundOperatorExpression &expr, unique_ptr<Expression> *expr_ptr);
	virtual unique_ptr<Expression> VisitReplace(BoundReferenceExpression &expr, unique_ptr<Expression> *expr_ptr);
	virtual unique_ptr<Expression> VisitReplace(BoundSubqueryExpression &expr, unique_ptr<Expression> *expr_ptr);
	virtual unique_ptr<Expression> VisitReplace(BoundParameterExpression &expr, unique_ptr<Expression> *expr_ptr);
	virtual unique_ptr<Expression> VisitReplace(BoundWindowExpression &expr, unique_ptr<Expression> *expr_ptr);
	virtual unique_ptr<Expression> VisitReplace(BoundUnnestExpression &expr, unique_ptr<Expression> *expr_ptr);
};
} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/storage/statistics/distinct_statistics.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/types/hyperloglog.hpp
//
//
//===----------------------------------------------------------------------===//







// LICENSE_CHANGE_BEGIN
// The following code up to LICENSE_CHANGE_END is subject to THIRD PARTY LICENSE #1
// See the end of this file for a list

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// third_party/hyperloglog/hyperloglog.hpp
//
//
//===----------------------------------------------------------------------===//



#include <stdint.h>
#include <string.h>

namespace duckdb_hll {

/* Error codes */
#define HLL_C_OK  0
#define HLL_C_ERR -1

struct robj {
	void *ptr;
};

//! Create a new empty HyperLogLog object
robj *hll_create(void);
//! Convert hll from sparse to dense
int hllSparseToDense(robj *o);
//! Destroy the specified HyperLogLog object
void hll_destroy(robj *obj);
//! Add an element with the specified amount of bytes to the HyperLogLog. Returns C_ERR on failure, otherwise returns 0
//! if the cardinality did not change, and 1 otherwise.
int hll_add(robj *o, unsigned char *ele, size_t elesize);
//! Returns the estimated amount of unique elements seen by the HyperLogLog. Returns C_OK on success, or C_ERR on
//! failure.
int hll_count(robj *o, size_t *result);
//! Merge hll_count HyperLogLog objects into a single one. Returns NULL on failure, or the new HLL object on success.
robj *hll_merge(robj **hlls, size_t hll_count);
//! Get size (in bytes) of the HLL
uint64_t get_size();

uint64_t MurmurHash64A(const void *key, int len, unsigned int seed);

} // namespace duckdb_hll

namespace duckdb {

void AddToLogsInternal(UnifiedVectorFormat &vdata, idx_t count, uint64_t indices[], uint8_t counts[], void ***logs[],
                       const SelectionVector *log_sel);

void AddToSingleLogInternal(UnifiedVectorFormat &vdata, idx_t count, uint64_t indices[], uint8_t counts[], void *log);

} // namespace duckdb


// LICENSE_CHANGE_END


namespace duckdb_hll {
struct robj;
}

namespace duckdb {

enum class HLLStorageType { UNCOMPRESSED = 1 };

class FieldWriter;
class FieldReader;

//! The HyperLogLog class holds a HyperLogLog counter for approximate cardinality counting
class HyperLogLog {
public:
	HyperLogLog();
	~HyperLogLog();
	// implicit copying of HyperLogLog is not allowed
	HyperLogLog(const HyperLogLog &) = delete;

	//! Adds an element of the specified size to the HyperLogLog counter
	void Add(data_ptr_t element, idx_t size);
	//! Return the count of this HyperLogLog counter
	idx_t Count() const;
	//! Merge this HyperLogLog counter with another counter to create a new one
	unique_ptr<HyperLogLog> Merge(HyperLogLog &other);
	HyperLogLog *MergePointer(HyperLogLog &other);
	//! Merge a set of HyperLogLogs to create one big one
	static unique_ptr<HyperLogLog> Merge(HyperLogLog logs[], idx_t count);
	//! Get the size (in bytes) of a HLL
	static idx_t GetSize();
	//! Get pointer to the HLL
	data_ptr_t GetPtr() const;
	//! Get copy of the HLL
	unique_ptr<HyperLogLog> Copy();
	//! (De)Serialize the HLL
	void Serialize(FieldWriter &writer) const;
	static unique_ptr<HyperLogLog> Deserialize(FieldReader &reader);

public:
	//! Compute HLL hashes over vdata, and store them in 'hashes'
	//! Then, compute register indices and prefix lengths, and also store them in 'hashes' as a pair of uint32_t
	static void ProcessEntries(UnifiedVectorFormat &vdata, const LogicalType &type, uint64_t hashes[], uint8_t counts[],
	                           idx_t count);
	//! Add the indices and counts to the logs
	static void AddToLogs(UnifiedVectorFormat &vdata, idx_t count, uint64_t indices[], uint8_t counts[],
	                      HyperLogLog **logs[], const SelectionVector *log_sel);
	//! Add the indices and counts to THIS log
	void AddToLog(UnifiedVectorFormat &vdata, idx_t count, uint64_t indices[], uint8_t counts[]);

private:
	explicit HyperLogLog(duckdb_hll::robj *hll);

	duckdb_hll::robj *hll;
	mutex lock;
};
} // namespace duckdb



namespace duckdb {
class Serializer;
class Deserializer;
class Vector;

class DistinctStatistics {
public:
	DistinctStatistics();
	explicit DistinctStatistics(unique_ptr<HyperLogLog> log, idx_t sample_count, idx_t total_count);

	//! The HLL of the table
	unique_ptr<HyperLogLog> log;
	//! How many values have been sampled into the HLL
	atomic<idx_t> sample_count;
	//! How many values have been inserted (before sampling)
	atomic<idx_t> total_count;

public:
	void Merge(const DistinctStatistics &other);

	unique_ptr<DistinctStatistics> Copy() const;

	void Serialize(Serializer &serializer) const;
	void Serialize(FieldWriter &writer) const;

	static unique_ptr<DistinctStatistics> Deserialize(Deserializer &source);
	static unique_ptr<DistinctStatistics> Deserialize(FieldReader &reader);

	void Update(Vector &update, idx_t count, bool sample = true);
	void Update(UnifiedVectorFormat &update_data, const LogicalType &ptype, idx_t count, bool sample = true);

	string ToString() const;
	idx_t GetCount() const;

	static bool TypeIsSupported(const LogicalType &type);

private:
	//! For distinct statistics we sample the input to speed up insertions
	static constexpr const double SAMPLE_RATE = 0.1;
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/planner/table_filter.hpp
//
//
//===----------------------------------------------------------------------===//








namespace duckdb {
class BaseStatistics;
class FieldWriter;
class FieldReader;

enum class TableFilterType : uint8_t {
	CONSTANT_COMPARISON = 0, // constant comparison (e.g. =C, >C, >=C, <C, <=C)
	IS_NULL = 1,
	IS_NOT_NULL = 2,
	CONJUNCTION_OR = 3,
	CONJUNCTION_AND = 4
};

//! TableFilter represents a filter pushed down into the table scan.
class TableFilter {
public:
	TableFilter(TableFilterType filter_type_p) : filter_type(filter_type_p) {
	}
	virtual ~TableFilter() {
	}

	TableFilterType filter_type;

public:
	//! Returns true if the statistics indicate that the segment can contain values that satisfy that filter
	virtual FilterPropagateResult CheckStatistics(BaseStatistics &stats) = 0;
	virtual string ToString(const string &column_name) = 0;
	virtual bool Equals(const TableFilter &other) const {
		return filter_type != other.filter_type;
	}

	void Serialize(Serializer &serializer) const;
	virtual void Serialize(FieldWriter &writer) const = 0;
	static unique_ptr<TableFilter> Deserialize(Deserializer &source);

public:
	template <class TARGET>
	TARGET &Cast() {
		if (filter_type != TARGET::TYPE) {
			throw InternalException("Failed to cast table to type - table filter type mismatch");
		}
		return reinterpret_cast<TARGET &>(*this);
	}

	template <class TARGET>
	const TARGET &Cast() const {
		if (filter_type != TARGET::TYPE) {
			throw InternalException("Failed to cast table to type - table filter type mismatch");
		}
		return reinterpret_cast<const TARGET &>(*this);
	}
};

class TableFilterSet {
public:
	unordered_map<idx_t, unique_ptr<TableFilter>> filters;

public:
	void PushFilter(idx_t table_index, unique_ptr<TableFilter> filter);

	bool Equals(TableFilterSet &other) {
		if (filters.size() != other.filters.size()) {
			return false;
		}
		for (auto &entry : filters) {
			auto other_entry = other.filters.find(entry.first);
			if (other_entry == other.filters.end()) {
				return false;
			}
			if (!entry.second->Equals(*other_entry->second)) {
				return false;
			}
		}
		return true;
	}
	static bool Equals(TableFilterSet *left, TableFilterSet *right) {
		if (left == right) {
			return true;
		}
		if (!left || !right) {
			return false;
		}
		return left->Equals(*right);
	}

	void Serialize(Serializer &serializer) const;
	static unique_ptr<TableFilterSet> Deserialize(Deserializer &source);
};

} // namespace duckdb


namespace duckdb {

class EstimatedProperties {
public:
	EstimatedProperties(double cardinality, double cost) : cardinality(cardinality), cost(cost) {};
	EstimatedProperties() : cardinality(0), cost(0) {};

	template <class T>
	T GetCardinality() const {
		throw NotImplementedException("Unsupported type for GetCardinality");
	}
	double GetCost() const;
	void SetCost(double new_cost);
	void SetCardinality(double cardinality);

private:
	double cardinality;
	double cost;

public:
	unique_ptr<EstimatedProperties> Copy();
};

template <>
double EstimatedProperties::GetCardinality() const;

template <>
idx_t EstimatedProperties::GetCardinality() const;

} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/optimizer/join_order/query_graph.hpp
//
//
//===----------------------------------------------------------------------===//









//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/planner/column_binding.hpp
//
//
//===----------------------------------------------------------------------===//






#include <functional>

namespace duckdb {

struct ColumnBinding {
	idx_t table_index;
	// This index is local to a Binding, and has no meaning outside of the context of the Binding that created it
	idx_t column_index;

	ColumnBinding() : table_index(DConstants::INVALID_INDEX), column_index(DConstants::INVALID_INDEX) {
	}
	ColumnBinding(idx_t table, idx_t column) : table_index(table), column_index(column) {
	}

	string ToString() const {
		return "#[" + to_string(table_index) + "." + to_string(column_index) + "]";
	}

	bool operator==(const ColumnBinding &rhs) const {
		return table_index == rhs.table_index && column_index == rhs.column_index;
	}
};

} // namespace duckdb



#include <functional>

namespace duckdb {
class Expression;
class LogicalOperator;

struct FilterInfo {
	FilterInfo(JoinRelationSet &set, idx_t filter_index) : set(set), filter_index(filter_index) {
	}

	JoinRelationSet &set;
	idx_t filter_index;
	optional_ptr<JoinRelationSet> left_set;
	optional_ptr<JoinRelationSet> right_set;
	ColumnBinding left_binding;
	ColumnBinding right_binding;
};

struct FilterNode {
	vector<reference<FilterInfo>> filters;
	unordered_map<idx_t, unique_ptr<FilterNode>> children;
};

struct NeighborInfo {
	NeighborInfo(JoinRelationSet &neighbor) : neighbor(neighbor) {
	}

	JoinRelationSet &neighbor;
	vector<reference<FilterInfo>> filters;
};

//! The QueryGraph contains edges between relations and allows edges to be created/queried
class QueryGraph {
public:
	//! Contains a node with info about neighboring relations and child edge infos
	struct QueryEdge {
		vector<unique_ptr<NeighborInfo>> neighbors;
		unordered_map<idx_t, unique_ptr<QueryEdge>> children;
	};

public:
	string ToString() const;
	void Print();

	//! Create an edge in the edge_set
	void CreateEdge(JoinRelationSet &left, JoinRelationSet &right, optional_ptr<FilterInfo> info);
	//! Returns a connection if there is an edge that connects these two sets, or nullptr otherwise
	vector<reference<NeighborInfo>> GetConnections(JoinRelationSet &node, JoinRelationSet &other);
	//! Enumerate the neighbors of a specific node that do not belong to any of the exclusion_set. Note that if a
	//! neighbor has multiple nodes, this function will return the lowest entry in that set.
	vector<idx_t> GetNeighbors(JoinRelationSet &node, unordered_set<idx_t> &exclusion_set);
	//! Enumerate all neighbors of a given JoinRelationSet node
	void EnumerateNeighbors(JoinRelationSet &node, const std::function<bool(NeighborInfo &)> &callback);

private:
	//! Get the QueryEdge of a specific node
	QueryEdge &GetQueryEdge(JoinRelationSet &left);

	QueryEdge root;
};

} // namespace duckdb






namespace duckdb {

class JoinOrderOptimizer;

class JoinNode {
public:
	//! Represents a node in the join plan
	JoinRelationSet &set;
	optional_ptr<NeighborInfo> info;
	//! If the JoinNode is a base table, then base_cardinality is the cardinality before filters
	//! estimated_props.cardinality will be the cardinality after filters. With no filters, the two are equal
	bool has_filter;
	optional_ptr<JoinNode> left;
	optional_ptr<JoinNode> right;

	unique_ptr<EstimatedProperties> estimated_props;

	//! Create a leaf node in the join tree
	//! set cost to 0 for leaf nodes
	//! cost will be the cost to *produce* an intermediate table
	JoinNode(JoinRelationSet &set, const double base_cardinality);

	//! Create an intermediate node in the join tree. base_cardinality = estimated_props.cardinality
	JoinNode(JoinRelationSet &set, optional_ptr<NeighborInfo> info, JoinNode &left, JoinNode &right,
	         const double base_cardinality, double cost);

	bool operator==(const JoinNode &other) {
		return other.set.ToString().compare(set.ToString()) == 0;
	}

private:
	double base_cardinality;

public:
	template <class CARDINALITY_TYPE>
	CARDINALITY_TYPE GetCardinality() const {
		return estimated_props->GetCardinality<CARDINALITY_TYPE>();
	}
	double GetCost();
	void SetCost(double cost);
	double GetBaseTableCardinality();
	void SetBaseTableCardinality(double base_card);
	void SetEstimatedCardinality(double estimated_card);
	void PrintJoinNode();
	string ToString();
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/optional_idx.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

class optional_idx {
	static constexpr const idx_t INVALID_INDEX = idx_t(-1);

public:
	optional_idx() : index(INVALID_INDEX) {
	}
	optional_idx(idx_t index) : index(index) { // NOLINT: allow implicit conversion from idx_t
		if (index == INVALID_INDEX) {
			throw InternalException("optional_idx cannot be initialized with an invalid index");
		}
	}

	static optional_idx Invalid() {
		return INVALID_INDEX;
	}

	bool IsValid() const {
		return index != DConstants::INVALID_INDEX;
	}
	idx_t GetIndex() {
		if (index == INVALID_INDEX) {
			throw InternalException("Attempting to get the index of an optional_idx that is not set");
		}
		return index;
	}

private:
	idx_t index;
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/execution/physical_operator_states.hpp
//
//
//===----------------------------------------------------------------------===//











namespace duckdb {
class Event;
class Executor;
class PhysicalOperator;
class Pipeline;
class PipelineBuildState;
class MetaPipeline;
class InterruptState;

struct SourcePartitionInfo {
	//! The current batch index
	//! This is only set in case RequiresBatchIndex() is true, and the source has support for it (SupportsBatchIndex())
	//! Otherwise this is left on INVALID_INDEX
	//! The batch index is a globally unique, increasing index that should be used to maintain insertion order
	//! //! in conjunction with parallelism
	optional_idx batch_index;
	//! The minimum batch index that any thread is currently actively reading
	optional_idx min_batch_index;
};

// LCOV_EXCL_START
class OperatorState {
public:
	virtual ~OperatorState() {
	}

	virtual void Finalize(const PhysicalOperator &op, ExecutionContext &context) {
	}

	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}
	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
};

class GlobalOperatorState {
public:
	virtual ~GlobalOperatorState() {
	}

	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}
	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
};

class GlobalSinkState {
public:
	GlobalSinkState() : state(SinkFinalizeType::READY) {
	}
	virtual ~GlobalSinkState() {
	}

	SinkFinalizeType state;

	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}
	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
};

class LocalSinkState {
public:
	virtual ~LocalSinkState() {
	}

	//! Source partition info
	SourcePartitionInfo partition_info;

	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}
	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
};

class GlobalSourceState {
public:
	virtual ~GlobalSourceState() {
	}

	virtual idx_t MaxThreads() {
		return 1;
	}

	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}
	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
};

class LocalSourceState {
public:
	virtual ~LocalSourceState() {
	}

	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}
	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
};

struct OperatorSinkInput {
	GlobalSinkState &global_state;
	LocalSinkState &local_state;
	InterruptState &interrupt_state;
};

struct OperatorSourceInput {
	GlobalSourceState &global_state;
	LocalSourceState &local_state;
	InterruptState &interrupt_state;
};

// LCOV_EXCL_STOP

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// src/include/duckdb/common/enums/order_preservation_type.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

//===--------------------------------------------------------------------===//
// Order Preservation Type
//===--------------------------------------------------------------------===//
enum class OrderPreservationType : uint8_t {
	NO_ORDER,        // the operator makes no guarantees on order preservation (i.e. it might re-order the entire input)
	INSERTION_ORDER, // the operator maintains the order of the child operators
	FIXED_ORDER      // the operator outputs rows in a fixed order that must be maintained (e.g. ORDER BY)
};

} // namespace duckdb


namespace duckdb {
class Event;
class Executor;
class PhysicalOperator;
class Pipeline;
class PipelineBuildState;
class MetaPipeline;

//! PhysicalOperator is the base class of the physical operators present in the
//! execution plan
class PhysicalOperator {
public:
	static constexpr const PhysicalOperatorType TYPE = PhysicalOperatorType::INVALID;

public:
	PhysicalOperator(PhysicalOperatorType type, vector<LogicalType> types, idx_t estimated_cardinality)
	    : type(type), types(std::move(types)), estimated_cardinality(estimated_cardinality) {
		estimated_props = make_uniq<EstimatedProperties>(estimated_cardinality, 0);
	}

	virtual ~PhysicalOperator() {
	}

	//! The physical operator type
	PhysicalOperatorType type;
	//! The set of children of the operator
	vector<unique_ptr<PhysicalOperator>> children;
	//! The types returned by this physical operator
	vector<LogicalType> types;
	//! The estimated cardinality of this physical operator
	idx_t estimated_cardinality;
	unique_ptr<EstimatedProperties> estimated_props;

	//! The global sink state of this operator
	unique_ptr<GlobalSinkState> sink_state;
	//! The global state of this operator
	unique_ptr<GlobalOperatorState> op_state;
	//! Lock for (re)setting any of the operator states
	mutex lock;

public:
	virtual string GetName() const;
	virtual string ParamsToString() const {
		return "";
	}
	virtual string ToString() const;
	void Print() const;
	virtual vector<const_reference<PhysicalOperator>> GetChildren() const;

	//! Return a vector of the types that will be returned by this operator
	const vector<LogicalType> &GetTypes() const {
		return types;
	}

	virtual bool Equals(const PhysicalOperator &other) const {
		return false;
	}

	virtual void Verify();

public:
	// Operator interface
	virtual unique_ptr<OperatorState> GetOperatorState(ExecutionContext &context) const;
	virtual unique_ptr<GlobalOperatorState> GetGlobalOperatorState(ClientContext &context) const;
	virtual OperatorResultType Execute(ExecutionContext &context, DataChunk &input, DataChunk &chunk,
	                                   GlobalOperatorState &gstate, OperatorState &state) const;
	virtual OperatorFinalizeResultType FinalExecute(ExecutionContext &context, DataChunk &chunk,
	                                                GlobalOperatorState &gstate, OperatorState &state) const;

	virtual bool ParallelOperator() const {
		return false;
	}

	virtual bool RequiresFinalExecute() const {
		return false;
	}

	//! The influence the operator has on order (insertion order means no influence)
	virtual OrderPreservationType OperatorOrder() const {
		return OrderPreservationType::INSERTION_ORDER;
	}

public:
	// Source interface
	virtual unique_ptr<LocalSourceState> GetLocalSourceState(ExecutionContext &context,
	                                                         GlobalSourceState &gstate) const;
	virtual unique_ptr<GlobalSourceState> GetGlobalSourceState(ClientContext &context) const;
	virtual SourceResultType GetData(ExecutionContext &context, DataChunk &chunk, OperatorSourceInput &input) const;

	virtual idx_t GetBatchIndex(ExecutionContext &context, DataChunk &chunk, GlobalSourceState &gstate,
	                            LocalSourceState &lstate) const;

	virtual bool IsSource() const {
		return false;
	}

	virtual bool ParallelSource() const {
		return false;
	}

	virtual bool SupportsBatchIndex() const {
		return false;
	}

	//! The type of order emitted by the operator (as a source)
	virtual OrderPreservationType SourceOrder() const {
		return OrderPreservationType::INSERTION_ORDER;
	}

	//! Returns the current progress percentage, or a negative value if progress bars are not supported
	virtual double GetProgress(ClientContext &context, GlobalSourceState &gstate) const;

public:
	// Sink interface

	//! The sink method is called constantly with new input, as long as new input is available. Note that this method
	//! CAN be called in parallel, proper locking is needed when accessing data inside the GlobalSinkState.
	virtual SinkResultType Sink(ExecutionContext &context, DataChunk &chunk, OperatorSinkInput &input) const;
	// The combine is called when a single thread has completed execution of its part of the pipeline, it is the final
	// time that a specific LocalSinkState is accessible. This method can be called in parallel while other Sink() or
	// Combine() calls are active on the same GlobalSinkState.
	virtual void Combine(ExecutionContext &context, GlobalSinkState &gstate, LocalSinkState &lstate) const;
	//! The finalize is called when ALL threads are finished execution. It is called only once per pipeline, and is
	//! entirely single threaded.
	//! If Finalize returns SinkResultType::FINISHED, the sink is marked as finished
	virtual SinkFinalizeType Finalize(Pipeline &pipeline, Event &event, ClientContext &context,
	                                  GlobalSinkState &gstate) const;
	//! For sinks with RequiresBatchIndex set to true, when a new batch starts being processed this method is called
	//! This allows flushing of the current batch (e.g. to disk)
	virtual void NextBatch(ExecutionContext &context, GlobalSinkState &state, LocalSinkState &lstate_p) const;

	virtual unique_ptr<LocalSinkState> GetLocalSinkState(ExecutionContext &context) const;
	virtual unique_ptr<GlobalSinkState> GetGlobalSinkState(ClientContext &context) const;

	//! The maximum amount of memory the operator should use per thread.
	static idx_t GetMaxThreadMemory(ClientContext &context);

	virtual bool IsSink() const {
		return false;
	}

	virtual bool ParallelSink() const {
		return false;
	}

	virtual bool RequiresBatchIndex() const {
		return false;
	}

	//! Whether or not the sink operator depends on the order of the input chunks
	//! If this is set to true, we cannot do things like caching intermediate vectors
	virtual bool SinkOrderDependent() const {
		return false;
	}

public:
	// Pipeline construction
	virtual vector<const_reference<PhysicalOperator>> GetSources() const;
	bool AllSourcesSupportBatchIndex() const;

	virtual void BuildPipelines(Pipeline &current, MetaPipeline &meta_pipeline);

public:
	template <class TARGET>
	TARGET &Cast() {
		if (TARGET::TYPE != PhysicalOperatorType::INVALID && type != TARGET::TYPE) {
			throw InternalException("Failed to cast physical operator to type - physical operator type mismatch");
		}
		return reinterpret_cast<TARGET &>(*this);
	}

	template <class TARGET>
	const TARGET &Cast() const {
		if (TARGET::TYPE != PhysicalOperatorType::INVALID && type != TARGET::TYPE) {
			throw InternalException("Failed to cast physical operator to type - physical operator type mismatch");
		}
		return reinterpret_cast<const TARGET &>(*this);
	}
};

//! Contains state for the CachingPhysicalOperator
class CachingOperatorState : public OperatorState {
public:
	~CachingOperatorState() override {
	}

	void Finalize(const PhysicalOperator &op, ExecutionContext &context) override {
	}

	unique_ptr<DataChunk> cached_chunk;
	bool initialized = false;
	//! Whether or not the chunk can be cached
	bool can_cache_chunk = false;
};

//! Base class that caches output from child Operator class. Note that Operators inheriting from this class should also
//! inherit their state class from the CachingOperatorState.
class CachingPhysicalOperator : public PhysicalOperator {
public:
	static constexpr const idx_t CACHE_THRESHOLD = 64;
	CachingPhysicalOperator(PhysicalOperatorType type, vector<LogicalType> types, idx_t estimated_cardinality);

	bool caching_supported;

public:
	OperatorResultType Execute(ExecutionContext &context, DataChunk &input, DataChunk &chunk,
	                           GlobalOperatorState &gstate, OperatorState &state) const final;
	OperatorFinalizeResultType FinalExecute(ExecutionContext &context, DataChunk &chunk, GlobalOperatorState &gstate,
	                                        OperatorState &state) const final;

	bool RequiresFinalExecute() const final {
		return caching_supported;
	}

protected:
	//! Child classes need to implement the ExecuteInternal method instead of the Execute
	virtual OperatorResultType ExecuteInternal(ExecutionContext &context, DataChunk &input, DataChunk &chunk,
	                                           GlobalOperatorState &gstate, OperatorState &state) const = 0;

private:
	bool CanCacheType(const LogicalType &type);
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/function/table_function.hpp
//
//
//===----------------------------------------------------------------------===//







//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/planner/bind_context.hpp
//
//
//===----------------------------------------------------------------------===//






//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/expression/columnref_expression.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {

//! Represents a reference to a column from either the FROM clause or from an
//! alias
class ColumnRefExpression : public ParsedExpression {
public:
	static constexpr const ExpressionClass TYPE = ExpressionClass::COLUMN_REF;

public:
	//! Specify both the column and table name
	ColumnRefExpression(string column_name, string table_name);
	//! Only specify the column name, the table name will be derived later
	explicit ColumnRefExpression(string column_name);
	//! Specify a set of names
	explicit ColumnRefExpression(vector<string> column_names);

	//! The stack of names in order of which they appear (column_names[0].column_names[1].column_names[2]....)
	vector<string> column_names;

public:
	bool IsQualified() const;
	const string &GetColumnName() const;
	const string &GetTableName() const;
	bool IsScalar() const override {
		return false;
	}

	string GetName() const override;
	string ToString() const override;

	static bool Equal(const ColumnRefExpression &a, const ColumnRefExpression &b);
	hash_t Hash() const override;

	unique_ptr<ParsedExpression> Copy() const override;

	void Serialize(FieldWriter &writer) const override;
	static unique_ptr<ParsedExpression> Deserialize(ExpressionType type, FieldReader &source);
	void FormatSerialize(FormatSerializer &serializer) const override;
	static unique_ptr<ParsedExpression> FormatDeserialize(ExpressionType type, FormatDeserializer &deserializer);
};
} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/qualified_name_set.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/types/hash.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {

struct string_t;
struct interval_t;

// efficient hash function that maximizes the avalanche effect and minimizes
// bias
// see: https://nullprogram.com/blog/2018/07/31/

inline hash_t murmurhash64(uint64_t x) {
	x ^= x >> 32;
	x *= 0xd6e8feb86659fd93U;
	x ^= x >> 32;
	x *= 0xd6e8feb86659fd93U;
	x ^= x >> 32;
	return x;
}

inline hash_t murmurhash32(uint32_t x) {
	return murmurhash64(x);
}

template <class T>
hash_t Hash(T value) {
	return murmurhash32(value);
}

//! Combine two hashes by XORing them
inline hash_t CombineHash(hash_t left, hash_t right) {
	return left ^ right;
}

template <>
DUCKDB_API hash_t Hash(uint64_t val);
template <>
DUCKDB_API hash_t Hash(int64_t val);
template <>
DUCKDB_API hash_t Hash(hugeint_t val);
template <>
DUCKDB_API hash_t Hash(float val);
template <>
DUCKDB_API hash_t Hash(double val);
template <>
DUCKDB_API hash_t Hash(const char *val);
template <>
DUCKDB_API hash_t Hash(char *val);
template <>
DUCKDB_API hash_t Hash(string_t val);
template <>
DUCKDB_API hash_t Hash(interval_t val);
DUCKDB_API hash_t Hash(const char *val, size_t size);
DUCKDB_API hash_t Hash(uint8_t *val, size_t size);

} // namespace duckdb



namespace duckdb {

struct QualifiedColumnHashFunction {
	uint64_t operator()(const QualifiedColumnName &a) const {
		std::hash<std::string> str_hasher;
		return str_hasher(a.schema) ^ str_hasher(a.table) ^ str_hasher(a.column);
	}
};

struct QualifiedColumnEquality {
	bool operator()(const QualifiedColumnName &a, const QualifiedColumnName &b) const {
		return a.schema == b.schema && a.table == b.table && a.column == b.column;
	}
};

using qualified_column_set_t = unordered_set<QualifiedColumnName, QualifiedColumnHashFunction, QualifiedColumnEquality>;

} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/planner/expression_binder.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/expression/bound_expression.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/field_writer.hpp
//
//
//===----------------------------------------------------------------------===//





//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/serializer/buffered_serializer.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {

#define SERIALIZER_DEFAULT_SIZE 1024

struct BinaryData {
	unsafe_unique_array<data_t> data;
	idx_t size;
};

class BufferedSerializer : public Serializer {
public:
	//! Serializes to a buffer allocated by the serializer, will expand when
	//! writing past the initial threshold
	DUCKDB_API explicit BufferedSerializer(idx_t maximum_size = SERIALIZER_DEFAULT_SIZE);
	//! Serializes to a provided (owned) data pointer
	BufferedSerializer(unsafe_unique_array<data_t> data, idx_t size);
	BufferedSerializer(data_ptr_t data, idx_t size);

	idx_t maximum_size;
	data_ptr_t data;

	BinaryData blob;

public:
	void WriteData(const_data_ptr_t buffer, uint64_t write_size) override;

	//! Retrieves the data after the writing has been completed
	BinaryData GetData() {
		return std::move(blob);
	}

	void Reset() {
		blob.size = 0;
	}
};

} // namespace duckdb

#include <type_traits>

namespace duckdb {
class BufferedSerializer;

struct IndexWriteOperation {
	template <class SRC, class DST>
	static DST Operation(SRC input) {
		return input.index;
	}
};

class FieldWriter {
public:
	DUCKDB_API explicit FieldWriter(Serializer &serializer);
	DUCKDB_API ~FieldWriter();

public:
	template <class T>
	void WriteField(const T &element) {
		static_assert(std::is_trivially_destructible<T>(), "WriteField object must be trivially destructible");

		AddField();
		WriteData(const_data_ptr_cast(&element), sizeof(T));
	}

	//! Write a string with a length prefix
	void WriteString(const string &val) {
		WriteStringLen(const_data_ptr_cast(val.c_str()), val.size());
	}
	void WriteStringLen(const_data_ptr_t val, idx_t len) {
		AddField();
		Write<uint32_t>((uint32_t)len);
		if (len > 0) {
			WriteData(val, len);
		}
	}
	void WriteBlob(const_data_ptr_t val, idx_t len) {
		AddField();
		if (len > 0) {
			WriteData(val, len);
		}
	}

	template <class T, class CONTAINER_TYPE = vector<T>>
	void WriteList(const CONTAINER_TYPE &elements) {
		AddField();
		Write<uint32_t>(elements.size());
		for (auto &element : elements) {
			Write<T>(element);
		}
	}

	template <class T, class SRC, class OP, class CONTAINER_TYPE = vector<SRC>>
	void WriteGenericList(const CONTAINER_TYPE &elements) {
		AddField();
		Write<uint32_t>(elements.size());
		for (auto &element : elements) {
			Write<T>(OP::template Operation<SRC, T>(element));
		}
	}

	template <class T>
	void WriteIndexList(const vector<T> &elements) {
		WriteGenericList<idx_t, T, IndexWriteOperation>(elements);
	}

	// vector<bool> yay
	template <class T, class CONTAINER_TYPE = vector<T>>
	void WriteListNoReference(const CONTAINER_TYPE &elements) {
		AddField();
		Write<uint32_t>(elements.size());
		for (auto element : elements) {
			Write<T>(element);
		}
	}

	template <class T>
	void WriteSerializable(const T &element) {
		AddField();
		element.Serialize(*buffer);
	}

	template <class T>
	void WriteSerializableList(const vector<unique_ptr<T>> &elements) {
		AddField();
		Write<uint32_t>(elements.size());
		for (idx_t i = 0; i < elements.size(); i++) {
			elements[i]->Serialize(*buffer);
		}
	}

	template <class T>
	void WriteRegularSerializableList(const vector<T> &elements) {
		AddField();
		Write<uint32_t>(elements.size());
		for (idx_t i = 0; i < elements.size(); i++) {
			elements[i].Serialize(*buffer);
		}
	}

	template <class T>
	void WriteOptional(const unique_ptr<T> &element) {
		AddField();
		Write<bool>(element ? true : false);
		if (element) {
			element->Serialize(*buffer);
		}
	}

	// Called after all fields have been written. Should always be called.
	DUCKDB_API void Finalize();

	Serializer &GetSerializer() {
		return *buffer;
	}

	void AddField() {
		field_count++;
	}

private:
	template <class T>
	void Write(const T &element) {
		WriteData(const_data_ptr_cast(&element), sizeof(T));
	}

	DUCKDB_API void WriteData(const_data_ptr_t buffer, idx_t write_size);

private:
	Serializer &serializer;
	unique_ptr<BufferedSerializer> buffer;
	idx_t field_count;
	bool finalized;
};

template <>
DUCKDB_API void FieldWriter::Write(const string &val);

class FieldDeserializer : public Deserializer {
public:
	explicit FieldDeserializer(Deserializer &root);

public:
	void ReadData(data_ptr_t buffer, idx_t read_size) override;

	void SetRemainingData(idx_t remaining_data);
	idx_t RemainingData();
	Deserializer &GetRoot() {
		return root;
	}

	ClientContext &GetContext() override {
		return root.GetContext();
	}

	optional_ptr<Catalog> GetCatalog() override {
		return root.GetCatalog();
	}

private:
	Deserializer &root;
	idx_t remaining_data;
};

struct IndexReadOperation {
	template <class SRC, class DST>
	static DST Operation(SRC input) {
		return DST(input);
	}
};

class FieldReader {
public:
	DUCKDB_API explicit FieldReader(Deserializer &source);
	DUCKDB_API ~FieldReader();

public:
	template <class T>
	T ReadRequired() {
		if (field_count >= max_field_count) {
			// field is not there, throw an exception
			throw SerializationException("Attempting to read a required field, but field is missing");
		}
		// field is there, read the actual value
		AddField();
		return source.Read<T>();
	}

	template <class T>
	T ReadField(T default_value) {
		if (field_count >= max_field_count) {
			// field is not there, read the default value
			return default_value;
		}
		// field is there, read the actual value
		AddField();
		return source.Read<T>();
	}

	template <class T, class CONTAINER_TYPE = vector<T>>
	bool ReadList(CONTAINER_TYPE &result) {
		if (field_count >= max_field_count) {
			// field is not there, return false and leave the result empty
			return false;
		}
		AddField();
		auto result_count = source.Read<uint32_t>();
		result.reserve(result_count);
		for (idx_t i = 0; i < result_count; i++) {
			result.push_back(source.Read<T>());
		}
		return true;
	}

	template <class T, class CONTAINER_TYPE = vector<T>>
	CONTAINER_TYPE ReadRequiredList() {
		if (field_count >= max_field_count) {
			// field is not there, throw an exception
			throw SerializationException("Attempting to read a required field, but field is missing");
		}
		AddField();
		auto result_count = source.Read<uint32_t>();
		CONTAINER_TYPE result;
		result.reserve(result_count);
		for (idx_t i = 0; i < result_count; i++) {
			result.push_back(source.Read<T>());
		}
		return result;
	}

	template <class T, class SRC, class OP>
	vector<T> ReadRequiredGenericList() {
		if (field_count >= max_field_count) {
			// field is not there, throw an exception
			throw SerializationException("Attempting to read a required field, but field is missing");
		}
		AddField();
		auto result_count = source.Read<uint32_t>();
		vector<T> result;
		result.reserve(result_count);
		for (idx_t i = 0; i < result_count; i++) {
			result.push_back(OP::template Operation<SRC, T>(source.Read<SRC>()));
		}
		return result;
	}

	template <class T>
	vector<T> ReadRequiredIndexList() {
		return ReadRequiredGenericList<T, idx_t, IndexReadOperation>();
	}

	template <class T, class CONTAINER_TYPE = set<T>>
	CONTAINER_TYPE ReadRequiredSet() {
		if (field_count >= max_field_count) {
			// field is not there, throw an exception
			throw SerializationException("Attempting to read a required field, but field is missing");
		}
		AddField();
		auto result_count = source.Read<uint32_t>();
		CONTAINER_TYPE result;
		for (idx_t i = 0; i < result_count; i++) {
			result.insert(source.Read<T>());
		}
		return result;
	}

	template <class T, typename... ARGS>
	unique_ptr<T> ReadOptional(unique_ptr<T> default_value, ARGS &&... args) {
		if (field_count >= max_field_count) {
			// field is not there, read the default value
			return default_value;
		}
		// field is there, read the actual value
		AddField();
		return source.template ReadOptional<T>(std::forward<ARGS>(args)...);
	}

	template <class T, class RETURN_TYPE = unique_ptr<T>>
	RETURN_TYPE ReadSerializable(RETURN_TYPE default_value) {
		if (field_count >= max_field_count) {
			// field is not there, read the default value
			return default_value;
		}
		// field is there, read the actual value
		AddField();
		return T::Deserialize(source);
	}

	template <class T, class RETURN_TYPE = unique_ptr<T>, typename... ARGS>
	RETURN_TYPE ReadSerializable(RETURN_TYPE default_value, ARGS &&... args) {
		if (field_count >= max_field_count) {
			// field is not there, read the default value
			return default_value;
		}
		// field is there, read the actual value
		AddField();
		return T::Deserialize(source, std::forward<ARGS>(args)...);
	}

	template <class T, class RETURN_TYPE = unique_ptr<T>>
	RETURN_TYPE ReadRequiredSerializable() {
		if (field_count >= max_field_count) {
			// field is not there, throw an exception
			throw SerializationException("Attempting to read mandatory field, but field is missing");
		}
		// field is there, read the actual value
		AddField();
		return T::Deserialize(source);
	}

	template <class T, class RETURN_TYPE = unique_ptr<T>, typename... ARGS>
	RETURN_TYPE ReadRequiredSerializable(ARGS &&... args) {
		if (field_count >= max_field_count) {
			// field is not there, throw an exception
			throw SerializationException("Attempting to read mandatory field, but field is missing");
		}
		// field is there, read the actual value
		AddField();
		return T::Deserialize(source, std::forward<ARGS>(args)...);
	}

	template <class T, class RETURN_TYPE = unique_ptr<T>, typename... ARGS>
	vector<RETURN_TYPE> ReadRequiredSerializableList(ARGS &&... args) {
		if (field_count >= max_field_count) {
			// field is not there, throw an exception
			throw SerializationException("Attempting to read mandatory field, but field is missing");
		}
		// field is there, read the actual value
		AddField();
		auto result_count = source.Read<uint32_t>();

		vector<RETURN_TYPE> result;
		for (idx_t i = 0; i < result_count; i++) {
			result.push_back(T::Deserialize(source, std::forward<ARGS>(args)...));
		}
		return result;
	}

	void ReadBlob(data_ptr_t result, idx_t read_size) {
		if (field_count >= max_field_count) {
			// field is not there, throw an exception
			throw SerializationException("Attempting to read a required field, but field is missing");
		}
		// field is there, read the actual value
		AddField();
		source.ReadData(result, read_size);
	}

	//! Called after all fields have been read. Should always be called.
	DUCKDB_API void Finalize();

	Deserializer &GetSource() {
		return source;
	}

private:
	void AddField() {
		field_count++;
	}

private:
	FieldDeserializer source;
	idx_t field_count;
	idx_t max_field_count;
	idx_t total_size;
	bool finalized;
};

} // namespace duckdb




namespace duckdb {

//! BoundExpression is an intermediate dummy class used by the binder. It is a ParsedExpression but holds an Expression.
//! It represents a successfully bound expression. It is used in the Binder to prevent re-binding of already bound parts
//! when dealing with subqueries.
class BoundExpression : public ParsedExpression {
public:
	static constexpr const ExpressionClass TYPE = ExpressionClass::BOUND_EXPRESSION;

public:
	BoundExpression(unique_ptr<Expression> expr);

	unique_ptr<Expression> expr;

public:
	static unique_ptr<Expression> &GetExpression(ParsedExpression &expr);

	string ToString() const override;

	bool Equals(const BaseExpression &other) const override;
	hash_t Hash() const override;

	unique_ptr<ParsedExpression> Copy() const override;

	void Serialize(FieldWriter &writer) const override;

	void FormatSerialize(FormatSerializer &serializer) const override;
};

} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/tokens.hpp
//
//
//===----------------------------------------------------------------------===//



namespace duckdb {

//===--------------------------------------------------------------------===//
// Statements
//===--------------------------------------------------------------------===//
class SQLStatement;

class AlterStatement;
class AttachStatement;
class CallStatement;
class CopyStatement;
class CreateStatement;
class DetachStatement;
class DeleteStatement;
class DropStatement;
class ExtensionStatement;
class InsertStatement;
class SelectStatement;
class TransactionStatement;
class UpdateStatement;
class PrepareStatement;
class ExecuteStatement;
class PragmaStatement;
class ShowStatement;
class ExplainStatement;
class ExportStatement;
class VacuumStatement;
class RelationStatement;
class SetStatement;
class SetVariableStatement;
class ResetVariableStatement;
class LoadStatement;
class LogicalPlanStatement;
class MultiStatement;

//===--------------------------------------------------------------------===//
// Query Node
//===--------------------------------------------------------------------===//
class QueryNode;
class SelectNode;
class SetOperationNode;
class RecursiveCTENode;

//===--------------------------------------------------------------------===//
// Expressions
//===--------------------------------------------------------------------===//
class ParsedExpression;

class BetweenExpression;
class CaseExpression;
class CastExpression;
class CollateExpression;
class ColumnRefExpression;
class ComparisonExpression;
class ConjunctionExpression;
class ConstantExpression;
class DefaultExpression;
class FunctionExpression;
class LambdaExpression;
class OperatorExpression;
class ParameterExpression;
class PositionalReferenceExpression;
class StarExpression;
class SubqueryExpression;
class WindowExpression;

//===--------------------------------------------------------------------===//
// Constraints
//===--------------------------------------------------------------------===//
class Constraint;

class NotNullConstraint;
class CheckConstraint;
class UniqueConstraint;
class ForeignKeyConstraint;

//===--------------------------------------------------------------------===//
// TableRefs
//===--------------------------------------------------------------------===//
class TableRef;

class BaseTableRef;
class JoinRef;
class SubqueryRef;
class TableFunctionRef;
class EmptyTableRef;
class ExpressionListRef;
class PivotRef;

//===--------------------------------------------------------------------===//
// Other
//===--------------------------------------------------------------------===//
struct SampleOptions;

} // namespace duckdb




namespace duckdb {

class Binder;
class ClientContext;
class QueryNode;

class ScalarFunctionCatalogEntry;
class AggregateFunctionCatalogEntry;
class ScalarMacroCatalogEntry;
class CatalogEntry;
class SimpleFunction;

struct DummyBinding;

struct BoundColumnReferenceInfo {
	string name;
	idx_t query_location;
};

struct BindResult {
	BindResult() {
	}
	explicit BindResult(string error) : error(error) {
	}
	explicit BindResult(unique_ptr<Expression> expr) : expression(std::move(expr)) {
	}

	bool HasError() {
		return !error.empty();
	}

	unique_ptr<Expression> expression;
	string error;
};

class ExpressionBinder {
public:
	ExpressionBinder(Binder &binder, ClientContext &context, bool replace_binder = false);
	virtual ~ExpressionBinder();

	//! The target type that should result from the binder. If the result is not of this type, a cast to this type will
	//! be added. Defaults to INVALID.
	LogicalType target_type;

	optional_ptr<DummyBinding> macro_binding;
	optional_ptr<vector<DummyBinding>> lambda_bindings;

public:
	unique_ptr<Expression> Bind(unique_ptr<ParsedExpression> &expr, optional_ptr<LogicalType> result_type = nullptr,
	                            bool root_expression = true);

	//! Returns whether or not any columns have been bound by the expression binder
	bool HasBoundColumns() {
		return !bound_columns.empty();
	}
	const vector<BoundColumnReferenceInfo> &GetBoundColumns() {
		return bound_columns;
	}

	string Bind(unique_ptr<ParsedExpression> &expr, idx_t depth, bool root_expression = false);

	unique_ptr<ParsedExpression> CreateStructExtract(unique_ptr<ParsedExpression> base, string field_name);
	unique_ptr<ParsedExpression> CreateStructPack(ColumnRefExpression &colref);
	BindResult BindQualifiedColumnName(ColumnRefExpression &colref, const string &table_name);

	unique_ptr<ParsedExpression> QualifyColumnName(const string &column_name, string &error_message);
	unique_ptr<ParsedExpression> QualifyColumnName(ColumnRefExpression &colref, string &error_message);

	// Bind table names to ColumnRefExpressions
	void QualifyColumnNames(unique_ptr<ParsedExpression> &expr);
	static void QualifyColumnNames(Binder &binder, unique_ptr<ParsedExpression> &expr);

	static unique_ptr<Expression> PushCollation(ClientContext &context, unique_ptr<Expression> source,
	                                            const string &collation, bool equality_only = false);
	static void TestCollation(ClientContext &context, const string &collation);

	bool BindCorrelatedColumns(unique_ptr<ParsedExpression> &expr);

	void BindChild(unique_ptr<ParsedExpression> &expr, idx_t depth, string &error);
	static void ExtractCorrelatedExpressions(Binder &binder, Expression &expr);

	static bool ContainsNullType(const LogicalType &type);
	static LogicalType ExchangeNullType(const LogicalType &type);
	static bool ContainsType(const LogicalType &type, LogicalTypeId target);
	static LogicalType ExchangeType(const LogicalType &type, LogicalTypeId target, LogicalType new_type);

	virtual bool QualifyColumnAlias(const ColumnRefExpression &colref);

	//! Bind the given expresion. Unlike Bind(), this does *not* mute the given ParsedExpression.
	//! Exposed to be used from sub-binders that aren't subclasses of ExpressionBinder.
	virtual BindResult BindExpression(unique_ptr<ParsedExpression> &expr_ptr, idx_t depth,
	                                  bool root_expression = false);

	void ReplaceMacroParametersRecursive(unique_ptr<ParsedExpression> &expr);

protected:
	BindResult BindExpression(BetweenExpression &expr, idx_t depth);
	BindResult BindExpression(CaseExpression &expr, idx_t depth);
	BindResult BindExpression(CollateExpression &expr, idx_t depth);
	BindResult BindExpression(CastExpression &expr, idx_t depth);
	BindResult BindExpression(ColumnRefExpression &expr, idx_t depth);
	BindResult BindExpression(ComparisonExpression &expr, idx_t depth);
	BindResult BindExpression(ConjunctionExpression &expr, idx_t depth);
	BindResult BindExpression(ConstantExpression &expr, idx_t depth);
	BindResult BindExpression(FunctionExpression &expr, idx_t depth, unique_ptr<ParsedExpression> &expr_ptr);
	BindResult BindExpression(LambdaExpression &expr, idx_t depth, const bool is_lambda,
	                          const LogicalType &list_child_type);
	BindResult BindExpression(OperatorExpression &expr, idx_t depth);
	BindResult BindExpression(ParameterExpression &expr, idx_t depth);
	BindResult BindExpression(SubqueryExpression &expr, idx_t depth);
	BindResult BindPositionalReference(unique_ptr<ParsedExpression> &expr, idx_t depth, bool root_expression);

	void TransformCapturedLambdaColumn(unique_ptr<Expression> &original, unique_ptr<Expression> &replacement,
	                                   vector<unique_ptr<Expression>> &captures, LogicalType &list_child_type);
	void CaptureLambdaColumns(vector<unique_ptr<Expression>> &captures, LogicalType &list_child_type,
	                          unique_ptr<Expression> &expr);

	static unique_ptr<ParsedExpression> GetSQLValueFunction(const string &column_name);

protected:
	virtual BindResult BindGroupingFunction(OperatorExpression &op, idx_t depth);
	virtual BindResult BindFunction(FunctionExpression &expr, ScalarFunctionCatalogEntry &function, idx_t depth);
	virtual BindResult BindLambdaFunction(FunctionExpression &expr, ScalarFunctionCatalogEntry &function, idx_t depth);
	virtual BindResult BindAggregate(FunctionExpression &expr, AggregateFunctionCatalogEntry &function, idx_t depth);
	virtual BindResult BindUnnest(FunctionExpression &expr, idx_t depth, bool root_expression);
	virtual BindResult BindMacro(FunctionExpression &expr, ScalarMacroCatalogEntry &macro, idx_t depth,
	                             unique_ptr<ParsedExpression> &expr_ptr);

	virtual string UnsupportedAggregateMessage();
	virtual string UnsupportedUnnestMessage();

	Binder &binder;
	ClientContext &context;
	optional_ptr<ExpressionBinder> stored_binder;
	vector<BoundColumnReferenceInfo> bound_columns;
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/planner/table_binding.hpp
//
//
//===----------------------------------------------------------------------===//










namespace duckdb {
class BindContext;
class BoundQueryNode;
class ColumnRefExpression;
class SubqueryRef;
class LogicalGet;
class TableCatalogEntry;
class TableFunctionCatalogEntry;
class BoundTableFunction;
class StandardEntry;
struct ColumnBinding;

enum class BindingType { BASE, TABLE, DUMMY, CATALOG_ENTRY };

//! A Binding represents a binding to a table, table-producing function or subquery with a specified table index.
struct Binding {
	Binding(BindingType binding_type, const string &alias, vector<LogicalType> types, vector<string> names,
	        idx_t index);
	virtual ~Binding() = default;

	//! The type of Binding
	BindingType binding_type;
	//! The alias of the binding
	string alias;
	//! The table index of the binding
	idx_t index;
	//! The types of the bound columns
	vector<LogicalType> types;
	//! Column names of the subquery
	vector<string> names;
	//! Name -> index for the names
	case_insensitive_map_t<column_t> name_map;

public:
	bool TryGetBindingIndex(const string &column_name, column_t &column_index);
	column_t GetBindingIndex(const string &column_name);
	bool HasMatchingBinding(const string &column_name);
	virtual string ColumnNotFoundError(const string &column_name) const;
	virtual BindResult Bind(ColumnRefExpression &colref, idx_t depth);
	virtual optional_ptr<StandardEntry> GetStandardEntry();

public:
	template <class TARGET>
	TARGET &Cast() {
		if (binding_type != TARGET::TYPE) {
			throw InternalException("Failed to cast binding to type - binding type mismatch");
		}
		return reinterpret_cast<TARGET &>(*this);
	}

	template <class TARGET>
	const TARGET &Cast() const {
		if (binding_type != TARGET::TYPE) {
			throw InternalException("Failed to cast binding to type - binding type mismatch");
		}
		return reinterpret_cast<const TARGET &>(*this);
	}
};

struct EntryBinding : public Binding {
public:
	static constexpr const BindingType TYPE = BindingType::CATALOG_ENTRY;

public:
	EntryBinding(const string &alias, vector<LogicalType> types, vector<string> names, idx_t index,
	             StandardEntry &entry);
	StandardEntry &entry;

public:
	optional_ptr<StandardEntry> GetStandardEntry() override;
};

//! TableBinding is exactly like the Binding, except it keeps track of which columns were bound in the linked LogicalGet
//! node for projection pushdown purposes.
struct TableBinding : public Binding {
public:
	static constexpr const BindingType TYPE = BindingType::TABLE;

public:
	TableBinding(const string &alias, vector<LogicalType> types, vector<string> names,
	             vector<column_t> &bound_column_ids, optional_ptr<StandardEntry> entry, idx_t index,
	             bool add_row_id = false);

	//! A reference to the set of bound column ids
	vector<column_t> &bound_column_ids;
	//! The underlying catalog entry (if any)
	optional_ptr<StandardEntry> entry;

public:
	unique_ptr<ParsedExpression> ExpandGeneratedColumn(const string &column_name);
	BindResult Bind(ColumnRefExpression &colref, idx_t depth) override;
	optional_ptr<StandardEntry> GetStandardEntry() override;
	string ColumnNotFoundError(const string &column_name) const override;
	// These are columns that are present in the name_map, appearing in the order that they're bound
	const vector<column_t> &GetBoundColumnIds() const;

protected:
	ColumnBinding GetColumnBinding(column_t column_index);
};

//! DummyBinding is like the Binding, except the alias and index are set by default. Used for binding lambdas and macro
//! parameters.
struct DummyBinding : public Binding {
public:
	static constexpr const BindingType TYPE = BindingType::DUMMY;
	// NOTE: changing this string conflicts with the storage version
	static constexpr const char *DUMMY_NAME = "0_macro_parameters";

public:
	DummyBinding(vector<LogicalType> types_p, vector<string> names_p, string dummy_name_p);

	//! Arguments
	vector<unique_ptr<ParsedExpression>> *arguments;
	//! The name of the dummy binding
	string dummy_name;

public:
	BindResult Bind(ColumnRefExpression &colref, idx_t depth) override;
	BindResult Bind(ColumnRefExpression &colref, idx_t lambda_index, idx_t depth);

	//! Given the parameter colref, returns a copy of the argument that was supplied for this parameter
	unique_ptr<ParsedExpression> ParamToArg(ColumnRefExpression &colref);
};

} // namespace duckdb


namespace duckdb {
class Binder;
class LogicalGet;
class BoundQueryNode;

class StarExpression;

class TableCatalogEntry;
class TableFunctionCatalogEntry;

struct UsingColumnSet {
	string primary_binding;
	unordered_set<string> bindings;
};

//! The BindContext object keeps track of all the tables and columns that are
//! encountered during the binding process.
class BindContext {
public:
	//! Keep track of recursive CTE references
	case_insensitive_map_t<std::shared_ptr<idx_t>> cte_references;

public:
	//! Given a column name, find the matching table it belongs to. Throws an
	//! exception if no table has a column of the given name.
	string GetMatchingBinding(const string &column_name);
	//! Like GetMatchingBinding, but instead of throwing an error if multiple tables have the same binding it will
	//! return a list of all the matching ones
	unordered_set<string> GetMatchingBindings(const string &column_name);
	//! Like GetMatchingBindings, but returns the top 3 most similar bindings (in levenshtein distance) instead of the
	//! matching ones
	vector<string> GetSimilarBindings(const string &column_name);

	optional_ptr<Binding> GetCTEBinding(const string &ctename);
	//! Binds a column expression to the base table. Returns the bound expression
	//! or throws an exception if the column could not be bound.
	BindResult BindColumn(ColumnRefExpression &colref, idx_t depth);
	string BindColumn(PositionalReferenceExpression &ref, string &table_name, string &column_name);
	unique_ptr<ColumnRefExpression> PositionToColumn(PositionalReferenceExpression &ref);

	unique_ptr<ParsedExpression> ExpandGeneratedColumn(const string &table_name, const string &column_name);

	unique_ptr<ParsedExpression> CreateColumnReference(const string &table_name, const string &column_name);
	unique_ptr<ParsedExpression> CreateColumnReference(const string &schema_name, const string &table_name,
	                                                   const string &column_name);
	unique_ptr<ParsedExpression> CreateColumnReference(const string &catalog_name, const string &schema_name,
	                                                   const string &table_name, const string &column_name);

	//! Generate column expressions for all columns that are present in the
	//! referenced tables. This is used to resolve the * expression in a
	//! selection list.
	void GenerateAllColumnExpressions(StarExpression &expr, vector<unique_ptr<ParsedExpression>> &new_select_list);
	//! Check if the given (binding, column_name) is in the exclusion/replacement lists.
	//! Returns true if it is in one of these lists, and should therefore be skipped.
	bool CheckExclusionList(StarExpression &expr, const string &column_name,
	                        vector<unique_ptr<ParsedExpression>> &new_select_list,
	                        case_insensitive_set_t &excluded_columns);

	const vector<reference<Binding>> &GetBindingsList() {
		return bindings_list;
	}

	void GetTypesAndNames(vector<string> &result_names, vector<LogicalType> &result_types);

	//! Adds a base table with the given alias to the BindContext.
	void AddBaseTable(idx_t index, const string &alias, const vector<string> &names, const vector<LogicalType> &types,
	                  vector<column_t> &bound_column_ids, StandardEntry *entry, bool add_row_id = true);
	//! Adds a call to a table function with the given alias to the BindContext.
	void AddTableFunction(idx_t index, const string &alias, const vector<string> &names,
	                      const vector<LogicalType> &types, vector<column_t> &bound_column_ids, StandardEntry *entry);
	//! Adds a table view with a given alias to the BindContext.
	void AddView(idx_t index, const string &alias, SubqueryRef &ref, BoundQueryNode &subquery, ViewCatalogEntry *view);
	//! Adds a subquery with a given alias to the BindContext.
	void AddSubquery(idx_t index, const string &alias, SubqueryRef &ref, BoundQueryNode &subquery);
	//! Adds a subquery with a given alias to the BindContext.
	void AddSubquery(idx_t index, const string &alias, TableFunctionRef &ref, BoundQueryNode &subquery);
	//! Adds a binding to a catalog entry with a given alias to the BindContext.
	void AddEntryBinding(idx_t index, const string &alias, const vector<string> &names,
	                     const vector<LogicalType> &types, StandardEntry &entry);
	//! Adds a base table with the given alias to the BindContext.
	void AddGenericBinding(idx_t index, const string &alias, const vector<string> &names,
	                       const vector<LogicalType> &types);

	//! Adds a base table with the given alias to the CTE BindContext.
	//! We need this to correctly bind recursive CTEs with multiple references.
	void AddCTEBinding(idx_t index, const string &alias, const vector<string> &names, const vector<LogicalType> &types);

	//! Add an implicit join condition (e.g. USING (x))
	void AddUsingBinding(const string &column_name, UsingColumnSet &set);

	void AddUsingBindingSet(unique_ptr<UsingColumnSet> set);

	//! Returns any using column set for the given column name, or nullptr if there is none. On conflict (multiple using
	//! column sets with the same name) throw an exception.
	optional_ptr<UsingColumnSet> GetUsingBinding(const string &column_name);
	//! Returns any using column set for the given column name, or nullptr if there is none
	optional_ptr<UsingColumnSet> GetUsingBinding(const string &column_name, const string &binding_name);
	//! Erase a using binding from the set of using bindings
	void RemoveUsingBinding(const string &column_name, UsingColumnSet &set);
	//! Transfer a using binding from one bind context to this bind context
	void TransferUsingBinding(BindContext &current_context, optional_ptr<UsingColumnSet> current_set,
	                          UsingColumnSet &new_set, const string &binding, const string &using_column);

	//! Fetch the actual column name from the given binding, or throws if none exists
	//! This can be different from "column_name" because of case insensitivity
	//! (e.g. "column_name" might return "COLUMN_NAME")
	string GetActualColumnName(const string &binding, const string &column_name);

	case_insensitive_map_t<std::shared_ptr<Binding>> GetCTEBindings() {
		return cte_bindings;
	}
	void SetCTEBindings(case_insensitive_map_t<std::shared_ptr<Binding>> bindings) {
		cte_bindings = bindings;
	}

	//! Alias a set of column names for the specified table, using the original names if there are not enough aliases
	//! specified.
	static vector<string> AliasColumnNames(const string &table_name, const vector<string> &names,
	                                       const vector<string> &column_aliases);

	//! Add all the bindings from a BindContext to this BindContext. The other BindContext is destroyed in the process.
	void AddContext(BindContext other);
	//! For semi and anti joins we remove the binding context of the right table after binding the condition.
	void RemoveContext(vector<reference<Binding>> &other_bindings_list);

	//! Gets a binding of the specified name. Returns a nullptr and sets the out_error if the binding could not be
	//! found.
	optional_ptr<Binding> GetBinding(const string &name, string &out_error);

private:
	void AddBinding(const string &alias, unique_ptr<Binding> binding);

private:
	//! The set of bindings
	case_insensitive_map_t<unique_ptr<Binding>> bindings;
	//! The list of bindings in insertion order
	vector<reference<Binding>> bindings_list;
	//! The set of columns used in USING join conditions
	case_insensitive_map_t<reference_set_t<UsingColumnSet>> using_columns;
	//! Using column sets
	vector<unique_ptr<UsingColumnSet>> using_column_sets;

	//! The set of CTE bindings
	case_insensitive_map_t<std::shared_ptr<Binding>> cte_bindings;
};
} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/planner/logical_operator.hpp
//
//
//===----------------------------------------------------------------------===//












#include <algorithm>
#include <functional>

namespace duckdb {

class FieldWriter;
class FieldReader;

//! The current version of the plan serialization format. Exposed via by @Serializer & @Deserializer
//! to be used by various Operator to know what format to read and write.
extern const uint64_t PLAN_SERIALIZATION_VERSION;

//! LogicalOperator is the base class of the logical operators present in the
//! logical query tree
class LogicalOperator {
public:
	explicit LogicalOperator(LogicalOperatorType type);
	LogicalOperator(LogicalOperatorType type, vector<unique_ptr<Expression>> expressions);
	virtual ~LogicalOperator();

	//! The type of the logical operator
	LogicalOperatorType type;
	//! The set of children of the operator
	vector<unique_ptr<LogicalOperator>> children;
	//! The set of expressions contained within the operator, if any
	vector<unique_ptr<Expression>> expressions;
	//! The types returned by this logical operator. Set by calling LogicalOperator::ResolveTypes.
	vector<LogicalType> types;
	//! Estimated Cardinality
	idx_t estimated_cardinality;
	bool has_estimated_cardinality;

	unique_ptr<EstimatedProperties> estimated_props;

public:
	virtual vector<ColumnBinding> GetColumnBindings();
	static vector<ColumnBinding> GenerateColumnBindings(idx_t table_idx, idx_t column_count);
	static vector<LogicalType> MapTypes(const vector<LogicalType> &types, const vector<idx_t> &projection_map);
	static vector<ColumnBinding> MapBindings(const vector<ColumnBinding> &types, const vector<idx_t> &projection_map);

	//! Resolve the types of the logical operator and its children
	void ResolveOperatorTypes();

	virtual string GetName() const;
	virtual string ParamsToString() const;
	virtual string ToString() const;
	DUCKDB_API void Print();
	//! Debug method: verify that the integrity of expressions & child nodes are maintained
	virtual void Verify(ClientContext &context);

	void AddChild(unique_ptr<LogicalOperator> child);
	virtual idx_t EstimateCardinality(ClientContext &context);

	//! Serializes a LogicalOperator to a stand-alone binary blob
	void Serialize(Serializer &serializer) const;
	//! Serializes an LogicalOperator to a stand-alone binary blob
	virtual void Serialize(FieldWriter &writer) const = 0;

	static unique_ptr<LogicalOperator> Deserialize(Deserializer &deserializer, PlanDeserializationState &state);

	virtual unique_ptr<LogicalOperator> Copy(ClientContext &context) const;

	virtual bool RequireOptimizer() const {
		return true;
	}

	//! Allows LogicalOperators to opt out of serialization
	virtual bool SupportSerialization() const {
		return true;
	};

	//! Returns the set of table indexes of this operator
	virtual vector<idx_t> GetTableIndex() const;

protected:
	//! Resolve types for this specific operator
	virtual void ResolveTypes() = 0;

public:
	template <class TARGET>
	TARGET &Cast() {
		if (TARGET::TYPE != LogicalOperatorType::LOGICAL_INVALID && type != TARGET::TYPE) {
			throw InternalException("Failed to cast logical operator to type - logical operator type mismatch");
		}
		return reinterpret_cast<TARGET &>(*this);
	}

	template <class TARGET>
	const TARGET &Cast() const {
		if (TARGET::TYPE != LogicalOperatorType::LOGICAL_INVALID && type != TARGET::TYPE) {
			throw InternalException("Failed to cast logical operator to type - logical operator type mismatch");
		}
		return reinterpret_cast<const TARGET &>(*this);
	}
};
} // namespace duckdb



#include <functional>

namespace duckdb {

class BaseStatistics;
class DependencyList;
class LogicalGet;
class TableFilterSet;

struct TableFunctionInfo {
	DUCKDB_API virtual ~TableFunctionInfo();

	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}
	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
};

struct GlobalTableFunctionState {
public:
	// value returned from MaxThreads when as many threads as possible should be used
	constexpr static const int64_t MAX_THREADS = 999999999;

public:
	DUCKDB_API virtual ~GlobalTableFunctionState();

	virtual idx_t MaxThreads() const {
		return 1;
	}

	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}
	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
};

struct LocalTableFunctionState {
	DUCKDB_API virtual ~LocalTableFunctionState();

	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}
	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
};

struct TableFunctionBindInput {
	TableFunctionBindInput(vector<Value> &inputs, named_parameter_map_t &named_parameters,
	                       vector<LogicalType> &input_table_types, vector<string> &input_table_names,
	                       optional_ptr<TableFunctionInfo> info)
	    : inputs(inputs), named_parameters(named_parameters), input_table_types(input_table_types),
	      input_table_names(input_table_names), info(info) {
	}

	vector<Value> &inputs;
	named_parameter_map_t &named_parameters;
	vector<LogicalType> &input_table_types;
	vector<string> &input_table_names;
	optional_ptr<TableFunctionInfo> info;
};

struct TableFunctionInitInput {
	TableFunctionInitInput(optional_ptr<const FunctionData> bind_data_p, const vector<column_t> &column_ids_p,
	                       const vector<idx_t> &projection_ids_p, optional_ptr<TableFilterSet> filters_p)
	    : bind_data(bind_data_p), column_ids(column_ids_p), projection_ids(projection_ids_p), filters(filters_p) {
	}

	optional_ptr<const FunctionData> bind_data;
	const vector<column_t> &column_ids;
	const vector<idx_t> projection_ids;
	optional_ptr<TableFilterSet> filters;

	bool CanRemoveFilterColumns() const {
		if (projection_ids.empty()) {
			// Not set, can't remove filter columns
			return false;
		} else if (projection_ids.size() == column_ids.size()) {
			// Filter column is used in remainder of plan, can't remove
			return false;
		} else {
			// Less columns need to be projected out than that we scan
			return true;
		}
	}
};

struct TableFunctionInput {
public:
	TableFunctionInput(optional_ptr<const FunctionData> bind_data_p,
	                   optional_ptr<LocalTableFunctionState> local_state_p,
	                   optional_ptr<GlobalTableFunctionState> global_state_p)
	    : bind_data(bind_data_p), local_state(local_state_p), global_state(global_state_p) {
	}

public:
	optional_ptr<const FunctionData> bind_data;
	optional_ptr<LocalTableFunctionState> local_state;
	optional_ptr<GlobalTableFunctionState> global_state;
};

enum ScanType { TABLE, PARQUET };

struct BindInfo {
public:
	explicit BindInfo(ScanType type_p) : type(type_p) {};
	unordered_map<string, Value> options;
	ScanType type;
	void InsertOption(const string &name, Value value) {
		if (options.find(name) != options.end()) {
			throw InternalException("This option already exists");
		}
		options[name] = std::move(value);
	}
	template <class T>
	T GetOption(const string &name) {
		if (options.find(name) == options.end()) {
			throw InternalException("This option does not exist");
		}
		return options[name].GetValue<T>();
	}
	template <class T>
	vector<T> GetOptionList(const string &name) {
		if (options.find(name) == options.end()) {
			throw InternalException("This option does not exist");
		}
		auto option = options[name];
		if (option.type().id() != LogicalTypeId::LIST) {
			throw InternalException("This option is not a list");
		}
		vector<T> result;
		auto list_children = ListValue::GetChildren(option);
		for (auto &child : list_children) {
			result.emplace_back(child.GetValue<T>());
		}
		return result;
	}
};

typedef unique_ptr<FunctionData> (*table_function_bind_t)(ClientContext &context, TableFunctionBindInput &input,
                                                          vector<LogicalType> &return_types, vector<string> &names);
typedef unique_ptr<TableRef> (*table_function_bind_replace_t)(ClientContext &context, TableFunctionBindInput &input);
typedef unique_ptr<GlobalTableFunctionState> (*table_function_init_global_t)(ClientContext &context,
                                                                             TableFunctionInitInput &input);
typedef unique_ptr<LocalTableFunctionState> (*table_function_init_local_t)(ExecutionContext &context,
                                                                           TableFunctionInitInput &input,
                                                                           GlobalTableFunctionState *global_state);
typedef unique_ptr<BaseStatistics> (*table_statistics_t)(ClientContext &context, const FunctionData *bind_data,
                                                         column_t column_index);
typedef void (*table_function_t)(ClientContext &context, TableFunctionInput &data, DataChunk &output);
typedef OperatorResultType (*table_in_out_function_t)(ExecutionContext &context, TableFunctionInput &data,
                                                      DataChunk &input, DataChunk &output);
typedef OperatorFinalizeResultType (*table_in_out_function_final_t)(ExecutionContext &context, TableFunctionInput &data,
                                                                    DataChunk &output);
typedef idx_t (*table_function_get_batch_index_t)(ClientContext &context, const FunctionData *bind_data,
                                                  LocalTableFunctionState *local_state,
                                                  GlobalTableFunctionState *global_state);

typedef BindInfo (*table_function_get_bind_info)(const FunctionData *bind_data);

typedef double (*table_function_progress_t)(ClientContext &context, const FunctionData *bind_data,
                                            const GlobalTableFunctionState *global_state);
typedef void (*table_function_dependency_t)(DependencyList &dependencies, const FunctionData *bind_data);
typedef unique_ptr<NodeStatistics> (*table_function_cardinality_t)(ClientContext &context,
                                                                   const FunctionData *bind_data);
typedef void (*table_function_pushdown_complex_filter_t)(ClientContext &context, LogicalGet &get,
                                                         FunctionData *bind_data,
                                                         vector<unique_ptr<Expression>> &filters);
typedef string (*table_function_to_string_t)(const FunctionData *bind_data);

typedef void (*table_function_serialize_t)(FieldWriter &writer, const FunctionData *bind_data,
                                           const TableFunction &function);
typedef unique_ptr<FunctionData> (*table_function_deserialize_t)(PlanDeserializationState &context, FieldReader &reader,
                                                                 TableFunction &function);

class TableFunction : public SimpleNamedParameterFunction {
public:
	DUCKDB_API
	TableFunction(string name, vector<LogicalType> arguments, table_function_t function,
	              table_function_bind_t bind = nullptr, table_function_init_global_t init_global = nullptr,
	              table_function_init_local_t init_local = nullptr);
	DUCKDB_API
	TableFunction(const vector<LogicalType> &arguments, table_function_t function, table_function_bind_t bind = nullptr,
	              table_function_init_global_t init_global = nullptr, table_function_init_local_t init_local = nullptr);
	DUCKDB_API TableFunction();

	//! Bind function
	//! This function is used for determining the return type of a table producing function and returning bind data
	//! The returned FunctionData object should be constant and should not be changed during execution.
	table_function_bind_t bind;
	//! (Optional) Bind replace function
	//! This function is called before the regular bind function. It allows returning a TableRef will be used to
	//! to generate a logical plan that replaces the LogicalGet of a regularly bound TableFunction. The BindReplace can
	//! also return a nullptr to indicate a regular bind needs to be performed instead.
	table_function_bind_replace_t bind_replace;
	//! (Optional) global init function
	//! Initialize the global operator state of the function.
	//! The global operator state is used to keep track of the progress in the table function and is shared between
	//! all threads working on the table function.
	table_function_init_global_t init_global;
	//! (Optional) local init function
	//! Initialize the local operator state of the function.
	//! The local operator state is used to keep track of the progress in the table function and is thread-local.
	table_function_init_local_t init_local;
	//! The main function
	table_function_t function;
	//! The table in-out function (if this is an in-out function)
	table_in_out_function_t in_out_function;
	//! The table in-out final function (if this is an in-out function)
	table_in_out_function_final_t in_out_function_final;
	//! (Optional) statistics function
	//! Returns the statistics of a specified column
	table_statistics_t statistics;
	//! (Optional) dependency function
	//! Sets up which catalog entries this table function depend on
	table_function_dependency_t dependency;
	//! (Optional) cardinality function
	//! Returns the expected cardinality of this scan
	table_function_cardinality_t cardinality;
	//! (Optional) pushdown a set of arbitrary filter expressions, rather than only simple comparisons with a constant
	//! Any functions remaining in the expression list will be pushed as a regular filter after the scan
	table_function_pushdown_complex_filter_t pushdown_complex_filter;
	//! (Optional) function for rendering the operator to a string in profiling output
	table_function_to_string_t to_string;
	//! (Optional) return how much of the table we have scanned up to this point (% of the data)
	table_function_progress_t table_scan_progress;
	//! (Optional) returns the current batch index of the current scan operator
	table_function_get_batch_index_t get_batch_index;
	//! (Optional) returns the extra batch info, currently only used for the substrait extension
	table_function_get_bind_info get_batch_info;

	table_function_serialize_t serialize;
	table_function_deserialize_t deserialize;
	bool verify_serialization = true;

	//! Whether or not the table function supports projection pushdown. If not supported a projection will be added
	//! that filters out unused columns.
	bool projection_pushdown;
	//! Whether or not the table function supports filter pushdown. If not supported a filter will be added
	//! that applies the table filter directly.
	bool filter_pushdown;
	//! Whether or not the table function can immediately prune out filter columns that are unused in the remainder of
	//! the query plan, e.g., "SELECT i FROM tbl WHERE j = 42;" - j does not need to leave the table function at all
	bool filter_prune;
	//! Additional function info, passed to the bind
	shared_ptr<TableFunctionInfo> function_info;

	DUCKDB_API bool Equal(const TableFunction &rhs) const;
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parallel/task_scheduler.hpp
//
//
//===----------------------------------------------------------------------===//






//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parallel/task.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {
class ClientContext;
class Executor;
class Task;
class DatabaseInstance;
struct ProducerToken;

enum class TaskExecutionMode : uint8_t { PROCESS_ALL, PROCESS_PARTIAL };

enum class TaskExecutionResult : uint8_t { TASK_FINISHED, TASK_NOT_FINISHED, TASK_ERROR, TASK_BLOCKED };

//! Generic parallel task
class Task : public std::enable_shared_from_this<Task> {
public:
	virtual ~Task() {
	}

	//! Execute the task in the specified execution mode
	//! If mode is PROCESS_ALL, Execute should always finish processing and return TASK_FINISHED
	//! If mode is PROCESS_PARTIAL, Execute can return TASK_NOT_FINISHED, in which case Execute will be called again
	//! In case of an error, TASK_ERROR is returned
	//! In case the task has interrupted, BLOCKED is returned.
	virtual TaskExecutionResult Execute(TaskExecutionMode mode) = 0;

	//! Descheduling a task ensures the task is not executed, but remains available for rescheduling as long as
	//! required, generally until some code in an operator calls the InterruptState::Callback() method of a state of the
	//! InterruptMode::TASK mode.
	virtual void Deschedule() {
		throw InternalException("Cannot deschedule task of base Task class");
	};

	//! Ensures a task is rescheduled to the correct queue
	virtual void Reschedule() {
		throw InternalException("Cannot reschedule task of base Task class");
	}
};

//! Execute a task within an executor, including exception handling
//! This should be used within queries
class ExecutorTask : public Task {
public:
	ExecutorTask(Executor &executor);
	ExecutorTask(ClientContext &context);
	virtual ~ExecutorTask();

	void Deschedule() override;
	void Reschedule() override;

	Executor &executor;

public:
	virtual TaskExecutionResult ExecuteTask(TaskExecutionMode mode) = 0;
	TaskExecutionResult Execute(TaskExecutionMode mode) override;
};

} // namespace duckdb



namespace duckdb {

struct ConcurrentQueue;
struct QueueProducerToken;
class ClientContext;
class DatabaseInstance;
class TaskScheduler;

struct SchedulerThread;

struct ProducerToken {
	ProducerToken(TaskScheduler &scheduler, unique_ptr<QueueProducerToken> token);
	~ProducerToken();

	TaskScheduler &scheduler;
	unique_ptr<QueueProducerToken> token;
	mutex producer_lock;
};

//! The TaskScheduler is responsible for managing tasks and threads
class TaskScheduler {
	// timeout for semaphore wait, default 5ms
	constexpr static int64_t TASK_TIMEOUT_USECS = 5000;

public:
	TaskScheduler(DatabaseInstance &db);
	~TaskScheduler();

	DUCKDB_API static TaskScheduler &GetScheduler(ClientContext &context);
	DUCKDB_API static TaskScheduler &GetScheduler(DatabaseInstance &db);

	unique_ptr<ProducerToken> CreateProducer();
	//! Schedule a task to be executed by the task scheduler
	void ScheduleTask(ProducerToken &producer, shared_ptr<Task> task);
	//! Fetches a task from a specific producer, returns true if successful or false if no tasks were available
	bool GetTaskFromProducer(ProducerToken &token, shared_ptr<Task> &task);
	//! Run tasks forever until "marker" is set to false, "marker" must remain valid until the thread is joined
	void ExecuteForever(atomic<bool> *marker);
	//! Run tasks until `marker` is set to false, `max_tasks` have been completed, or until there are no more tasks
	//! available. Returns the number of tasks that were completed.
	idx_t ExecuteTasks(atomic<bool> *marker, idx_t max_tasks);
	//! Run tasks until `max_tasks` have been completed, or until there are no more tasks available
	void ExecuteTasks(idx_t max_tasks);

	//! Sets the amount of active threads executing tasks for the system; n-1 background threads will be launched.
	//! The main thread will also be used for execution
	void SetThreads(int32_t n);
	//! Returns the number of threads
	DUCKDB_API int32_t NumberOfThreads();

	//! Send signals to n threads, signalling for them to wake up and attempt to execute a task
	void Signal(idx_t n);

private:
	void SetThreadsInternal(int32_t n);

private:
	DatabaseInstance &db;
	//! The task queue
	unique_ptr<ConcurrentQueue> queue;
	//! Lock for modifying the thread count
	mutex thread_lock;
	//! The active background threads of the task scheduler
	vector<unique_ptr<SchedulerThread>> threads;
	//! Markers used by the various threads, if the markers are set to "false" the thread execution is stopped
	vector<unique_ptr<atomic<bool>>> markers;
};

} // namespace duckdb



namespace duckdb {

class Executor;
class Event;
class MetaPipeline;

class PipelineBuildState {
public:
	//! How much to increment batch indexes when multiple pipelines share the same source
	constexpr static idx_t BATCH_INCREMENT = 10000000000000;

public:
	//! Duplicate eliminated join scan dependencies
	reference_map_t<const PhysicalOperator, reference<Pipeline>> delim_join_dependencies;

public:
	void SetPipelineSource(Pipeline &pipeline, PhysicalOperator &op);
	void SetPipelineSink(Pipeline &pipeline, optional_ptr<PhysicalOperator> op, idx_t sink_pipeline_count);
	void SetPipelineOperators(Pipeline &pipeline, vector<reference<PhysicalOperator>> operators);
	void AddPipelineOperator(Pipeline &pipeline, PhysicalOperator &op);
	shared_ptr<Pipeline> CreateChildPipeline(Executor &executor, Pipeline &pipeline, PhysicalOperator &op);

	optional_ptr<PhysicalOperator> GetPipelineSource(Pipeline &pipeline);
	optional_ptr<PhysicalOperator> GetPipelineSink(Pipeline &pipeline);
	vector<reference<PhysicalOperator>> GetPipelineOperators(Pipeline &pipeline);
};

//! The Pipeline class represents an execution pipeline starting at a
class Pipeline : public std::enable_shared_from_this<Pipeline> {
	friend class Executor;
	friend class PipelineExecutor;
	friend class PipelineEvent;
	friend class PipelineFinishEvent;
	friend class PipelineBuildState;
	friend class MetaPipeline;

public:
	explicit Pipeline(Executor &execution_context);

	Executor &executor;

public:
	ClientContext &GetClientContext();

	void AddDependency(shared_ptr<Pipeline> &pipeline);

	void Ready();
	void Reset();
	void ResetSink();
	void ResetSource(bool force);
	void ClearSource();
	void Schedule(shared_ptr<Event> &event);

	//! Finalize this pipeline
	void Finalize(Event &event);

	string ToString() const;
	void Print() const;
	void PrintDependencies() const;

	//! Returns query progress
	bool GetProgress(double &current_percentage, idx_t &estimated_cardinality);

	//! Returns a list of all operators (including source and sink) involved in this pipeline
	vector<reference<PhysicalOperator>> GetOperators();
	vector<const_reference<PhysicalOperator>> GetOperators() const;

	optional_ptr<PhysicalOperator> GetSink() {
		return sink;
	}

	optional_ptr<PhysicalOperator> GetSource() {
		return source;
	}

	//! Returns whether any of the operators in the pipeline care about preserving order
	bool IsOrderDependent() const;

	//! Registers a new batch index for a pipeline executor - returns the current minimum batch index
	idx_t RegisterNewBatchIndex();

	//! Updates the batch index of a pipeline (and returns the new minimum batch index)
	idx_t UpdateBatchIndex(idx_t old_index, idx_t new_index);

private:
	//! Whether or not the pipeline has been readied
	bool ready;
	//! Whether or not the pipeline has been initialized
	atomic<bool> initialized;
	//! The source of this pipeline
	optional_ptr<PhysicalOperator> source;
	//! The chain of intermediate operators
	vector<reference<PhysicalOperator>> operators;
	//! The sink (i.e. destination) for data; this is e.g. a hash table to-be-built
	optional_ptr<PhysicalOperator> sink;

	//! The global source state
	unique_ptr<GlobalSourceState> source_state;

	//! The parent pipelines (i.e. pipelines that are dependent on this pipeline to finish)
	vector<weak_ptr<Pipeline>> parents;
	//! The dependencies of this pipeline
	vector<weak_ptr<Pipeline>> dependencies;

	//! The base batch index of this pipeline
	idx_t base_batch_index = 0;
	//! Lock for accessing the set of batch indexes
	mutex batch_lock;
	//! The set of batch indexes that are currently being processed
	//! Despite batch indexes being unique - this is a multiset
	//! The reason is that when we start a new pipeline we insert the current minimum batch index as a placeholder
	//! Which leads to duplicate entries in the set of active batch indexes
	multiset<idx_t> batch_indexes;

private:
	void ScheduleSequentialTask(shared_ptr<Event> &event);
	bool LaunchScanTasks(shared_ptr<Event> &event, idx_t max_threads);

	bool ScheduleParallel(shared_ptr<Event> &event);
};

} // namespace duckdb


namespace duckdb {
class ClientContext;
class DataChunk;
class PhysicalOperator;
class PipelineExecutor;
class OperatorState;
class QueryProfiler;
class ThreadContext;
class Task;

struct PipelineEventStack;
struct ProducerToken;
struct ScheduleEventData;

class Executor {
	friend class Pipeline;
	friend class PipelineTask;
	friend class PipelineBuildState;

public:
	explicit Executor(ClientContext &context);
	~Executor();

	ClientContext &context;

public:
	static Executor &Get(ClientContext &context);

	void Initialize(PhysicalOperator &physical_plan);
	void Initialize(unique_ptr<PhysicalOperator> physical_plan);

	void CancelTasks();
	PendingExecutionResult ExecuteTask();

	void Reset();

	vector<LogicalType> GetTypes();

	unique_ptr<DataChunk> FetchChunk();

	//! Push a new error
	void PushError(PreservedError exception);

	//! True if an error has been thrown
	bool HasError();
	//! Throw the exception that was pushed using PushError.
	//! Should only be called if HasError returns true
	void ThrowException();

	//! Work on tasks for this specific executor, until there are no tasks remaining
	void WorkOnTasks();

	//! Flush a thread context into the client context
	void Flush(ThreadContext &context);

	//! Reschedules a task that was blocked
	void RescheduleTask(shared_ptr<Task> &task);

	//! Add the task to be rescheduled
	void AddToBeRescheduled(shared_ptr<Task> &task);

	//! Returns the progress of the pipelines
	bool GetPipelinesProgress(double &current_progress);

	void CompletePipeline() {
		completed_pipelines++;
	}
	ProducerToken &GetToken() {
		return *producer;
	}
	void AddEvent(shared_ptr<Event> event);

	void AddRecursiveCTE(PhysicalOperator &rec_cte);
	void ReschedulePipelines(const vector<shared_ptr<MetaPipeline>> &pipelines, vector<shared_ptr<Event>> &events);

	//! Whether or not the root of the pipeline is a result collector object
	bool HasResultCollector();
	//! Returns the query result - can only be used if `HasResultCollector` returns true
	unique_ptr<QueryResult> GetResult();

	//! Returns true if all pipelines have been completed
	bool ExecutionIsFinished();

private:
	void InitializeInternal(PhysicalOperator &physical_plan);

	void ScheduleEvents(const vector<shared_ptr<MetaPipeline>> &meta_pipelines);
	static void ScheduleEventsInternal(ScheduleEventData &event_data);

	static void VerifyScheduledEvents(const ScheduleEventData &event_data);
	static void VerifyScheduledEventsInternal(const idx_t i, const vector<Event *> &vertices, vector<bool> &visited,
	                                          vector<bool> &recursion_stack);

	static void SchedulePipeline(const shared_ptr<MetaPipeline> &pipeline, ScheduleEventData &event_data);

	bool NextExecutor();

	shared_ptr<Pipeline> CreateChildPipeline(Pipeline &current, PhysicalOperator &op);

	void VerifyPipeline(Pipeline &pipeline);
	void VerifyPipelines();

private:
	optional_ptr<PhysicalOperator> physical_plan;
	unique_ptr<PhysicalOperator> owned_plan;

	mutex executor_lock;
	mutex error_lock;
	//! All pipelines of the query plan
	vector<shared_ptr<Pipeline>> pipelines;
	//! The root pipelines of the query
	vector<shared_ptr<Pipeline>> root_pipelines;
	//! The recursive CTE's in this query plan
	vector<reference<PhysicalOperator>> recursive_ctes;
	//! The pipeline executor for the root pipeline
	unique_ptr<PipelineExecutor> root_executor;
	//! The current root pipeline index
	idx_t root_pipeline_idx;
	//! The producer of this query
	unique_ptr<ProducerToken> producer;
	//! Exceptions that occurred during the execution of the current query
	vector<PreservedError> exceptions;
	//! List of events
	vector<shared_ptr<Event>> events;
	//! The query profiler
	shared_ptr<QueryProfiler> profiler;

	//! The amount of completed pipelines of the query
	atomic<idx_t> completed_pipelines;
	//! The total amount of pipelines in the query
	idx_t total_pipelines;
	//! Whether or not execution is cancelled
	bool cancelled;

	//! The last pending execution result (if any)
	PendingExecutionResult execution_result;
	//! The current task in process (if any)
	shared_ptr<Task> task;

	//! Task that have been descheduled
	unordered_map<Task *, shared_ptr<Task>> to_be_rescheduled_tasks;
};
} // namespace duckdb


namespace duckdb {
class ClientContext;
class ClientContextLock;
class PreparedStatementData;

class PendingQueryResult : public BaseQueryResult {
	friend class ClientContext;

public:
	static constexpr const QueryResultType TYPE = QueryResultType::PENDING_RESULT;

public:
	DUCKDB_API PendingQueryResult(shared_ptr<ClientContext> context, PreparedStatementData &statement,
	                              vector<LogicalType> types, bool allow_stream_result);
	DUCKDB_API explicit PendingQueryResult(PreservedError error_message);
	DUCKDB_API ~PendingQueryResult();

public:
	//! Executes a single task within the query, returning whether or not the query is ready.
	//! If this returns RESULT_READY, the Execute function can be called to obtain a pointer to the result.
	//! If this returns RESULT_NOT_READY, the ExecuteTask function should be called again.
	//! If this returns EXECUTION_ERROR, an error occurred during execution.
	//! The error message can be obtained by calling GetError() on the PendingQueryResult.
	DUCKDB_API PendingExecutionResult ExecuteTask();

	//! Returns the result of the query as an actual query result.
	//! This returns (mostly) instantly if ExecuteTask has been called until RESULT_READY was returned.
	DUCKDB_API unique_ptr<QueryResult> Execute();

	DUCKDB_API void Close();

private:
	shared_ptr<ClientContext> context;
	bool allow_stream_result;

private:
	void CheckExecutableInternal(ClientContextLock &lock);

	PendingExecutionResult ExecuteTaskInternal(ClientContextLock &lock);
	unique_ptr<QueryResult> ExecuteInternal(ClientContextLock &lock);
	unique_ptr<ClientContextLock> LockContext();
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/main/prepared_statement.hpp
//
//
//===----------------------------------------------------------------------===//









namespace duckdb {
class ClientContext;
class PreparedStatementData;

//! A prepared statement
class PreparedStatement {
public:
	//! Create a successfully prepared prepared statement object with the given name
	DUCKDB_API PreparedStatement(shared_ptr<ClientContext> context, shared_ptr<PreparedStatementData> data,
	                             string query, idx_t n_param, case_insensitive_map_t<idx_t> named_param_map);
	//! Create a prepared statement that was not successfully prepared
	DUCKDB_API explicit PreparedStatement(PreservedError error);

	DUCKDB_API ~PreparedStatement();

public:
	//! The client context this prepared statement belongs to
	shared_ptr<ClientContext> context;
	//! The prepared statement data
	shared_ptr<PreparedStatementData> data;
	//! The query that is being prepared
	string query;
	//! Whether or not the statement was successfully prepared
	bool success;
	//! The error message (if success = false)
	PreservedError error;
	//! The amount of bound parameters
	idx_t n_param;
	//! The (optional) named parameters
	case_insensitive_map_t<idx_t> named_param_map;

public:
	//! Returns the stored error message
	DUCKDB_API const string &GetError();
	//! Returns the stored error object
	DUCKDB_API PreservedError &GetErrorObject();
	//! Returns whether or not an error occurred
	DUCKDB_API bool HasError() const;
	//! Returns the number of columns in the result
	DUCKDB_API idx_t ColumnCount();
	//! Returns the statement type of the underlying prepared statement object
	DUCKDB_API StatementType GetStatementType();
	//! Returns the underlying statement properties
	DUCKDB_API StatementProperties GetStatementProperties();
	//! Returns the result SQL types of the prepared statement
	DUCKDB_API const vector<LogicalType> &GetTypes();
	//! Returns the result names of the prepared statement
	DUCKDB_API const vector<string> &GetNames();
	//! Returns the map of parameter index to the expected type of parameter
	DUCKDB_API vector<LogicalType> GetExpectedParameterTypes() const;

	//! Create a pending query result of the prepared statement with the given set of arguments
	template <typename... Args>
	unique_ptr<PendingQueryResult> PendingQuery(Args... args) {
		vector<Value> values;
		return PendingQueryRecursive(values, args...);
	}

	//! Execute the prepared statement with the given set of arguments
	template <typename... Args>
	unique_ptr<QueryResult> Execute(Args... args) {
		vector<Value> values;
		return ExecuteRecursive(values, args...);
	}

	//! Create a pending query result of the prepared statement with the given set of arguments
	DUCKDB_API unique_ptr<PendingQueryResult> PendingQuery(vector<Value> &values, bool allow_stream_result = true);

	//! Execute the prepared statement with the given set of values
	DUCKDB_API unique_ptr<QueryResult> Execute(vector<Value> &values, bool allow_stream_result = true);

private:
	unique_ptr<PendingQueryResult> PendingQueryRecursive(vector<Value> &values) {
		return PendingQuery(values);
	}

	template <typename T, typename... Args>
	unique_ptr<PendingQueryResult> PendingQueryRecursive(vector<Value> &values, T value, Args... args) {
		values.push_back(Value::CreateValue<T>(value));
		return PendingQueryRecursive(values, args...);
	}

	unique_ptr<QueryResult> ExecuteRecursive(vector<Value> &values) {
		return Execute(values);
	}

	template <typename T, typename... Args>
	unique_ptr<QueryResult> ExecuteRecursive(vector<Value> &values, T value, Args... args) {
		values.push_back(Value::CreateValue<T>(value));
		return ExecuteRecursive(values, args...);
	}
};

} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/main/relation.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/join_type.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

//===--------------------------------------------------------------------===//
// Join Types
//===--------------------------------------------------------------------===//
enum class JoinType : uint8_t {
	INVALID = 0, // invalid join type
	LEFT = 1,    // left
	RIGHT = 2,   // right
	INNER = 3,   // inner
	OUTER = 4,   // outer
	SEMI = 5,    // SEMI join returns left side row ONLY if it has a join partner, no duplicates
	ANTI = 6,    // ANTI join returns left side row ONLY if it has NO join partner, no duplicates
	MARK = 7,    // MARK join returns marker indicating whether or not there is a join partner (true), there is no join
	             // partner (false)
	SINGLE = 8   // SINGLE join is like LEFT OUTER JOIN, BUT returns at most one join partner per entry on the LEFT side
	             // (and NULL if no partner is found)
};

//! True if join is left or full outer join
bool IsLeftOuterJoin(JoinType type);

//! True if join is rght or full outer join
bool IsRightOuterJoin(JoinType type);

// **DEPRECATED**: Use EnumUtil directly instead.
string JoinTypeToString(JoinType type);

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/relation_type.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

//===--------------------------------------------------------------------===//
// Catalog Types
//===--------------------------------------------------------------------===//
enum class RelationType : uint8_t {
	INVALID_RELATION,
	TABLE_RELATION,
	PROJECTION_RELATION,
	FILTER_RELATION,
	EXPLAIN_RELATION,
	CROSS_PRODUCT_RELATION,
	JOIN_RELATION,
	AGGREGATE_RELATION,
	SET_OPERATION_RELATION,
	DISTINCT_RELATION,
	LIMIT_RELATION,
	ORDER_RELATION,
	CREATE_VIEW_RELATION,
	CREATE_TABLE_RELATION,
	INSERT_RELATION,
	VALUE_LIST_RELATION,
	DELETE_RELATION,
	UPDATE_RELATION,
	WRITE_CSV_RELATION,
	WRITE_PARQUET_RELATION,
	READ_CSV_RELATION,
	SUBQUERY_RELATION,
	TABLE_FUNCTION_RELATION,
	VIEW_RELATION,
	QUERY_RELATION
};

string RelationTypeToString(RelationType type);

} // namespace duckdb





//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/main/client_context.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/catalog/catalog_entry/schema_catalog_entry.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/catalog/catalog_set.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/catalog/default/default_generator.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {
class ClientContext;

class DefaultGenerator {
public:
	explicit DefaultGenerator(Catalog &catalog) : catalog(catalog), created_all_entries(false) {
	}
	virtual ~DefaultGenerator() {
	}

	Catalog &catalog;
	atomic<bool> created_all_entries;

public:
	//! Creates a default entry with the specified name, or returns nullptr if no such entry can be generated
	virtual unique_ptr<CatalogEntry> CreateDefaultEntry(ClientContext &context, const string &entry_name) = 0;
	//! Get a list of all default entries in the generator
	virtual vector<string> GetDefaultEntries() = 0;
};

} // namespace duckdb







//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/transaction/transaction.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/catalog/catalog_entry/sequence_catalog_entry.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/catalog/standard_entry.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {
class SchemaCatalogEntry;

//! A StandardEntry is a catalog entry that is a member of a schema
class StandardEntry : public InCatalogEntry {
public:
	StandardEntry(CatalogType type, SchemaCatalogEntry &schema, Catalog &catalog, string name)
	    : InCatalogEntry(type, catalog, name), schema(schema) {
	}
	~StandardEntry() override {
	}

	//! The schema the entry belongs to
	SchemaCatalogEntry &schema;

public:
	SchemaCatalogEntry &ParentSchema() override {
		return schema;
	}
};

} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/parsed_data/create_sequence_info.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/parsed_data/create_info.hpp
//
//
//===----------------------------------------------------------------------===//





//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/parsed_data/parse_info.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

struct ParseInfo {
	virtual ~ParseInfo() {
	}

public:
	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}

	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
};

} // namespace duckdb



namespace duckdb {
struct AlterInfo;

enum class OnCreateConflict : uint8_t {
	// Standard: throw error
	ERROR_ON_CONFLICT,
	// CREATE IF NOT EXISTS, silently do nothing on conflict
	IGNORE_ON_CONFLICT,
	// CREATE OR REPLACE
	REPLACE_ON_CONFLICT,
	// Update on conflict - only support for functions. Add a function overload if the function already exists.
	ALTER_ON_CONFLICT
};

struct CreateInfo : public ParseInfo {
	explicit CreateInfo(CatalogType type, string schema = DEFAULT_SCHEMA, string catalog_p = INVALID_CATALOG)
	    : type(type), catalog(std::move(catalog_p)), schema(schema), on_conflict(OnCreateConflict::ERROR_ON_CONFLICT),
	      temporary(false), internal(false) {
	}
	~CreateInfo() override {
	}

	//! The to-be-created catalog type
	CatalogType type;
	//! The catalog name of the entry
	string catalog;
	//! The schema name of the entry
	string schema;
	//! What to do on create conflict
	OnCreateConflict on_conflict;
	//! Whether or not the entry is temporary
	bool temporary;
	//! Whether or not the entry is an internal entry
	bool internal;
	//! The SQL string of the CREATE statement
	string sql;

protected:
	virtual void SerializeInternal(Serializer &) const = 0;

	void DeserializeBase(Deserializer &deserializer);

public:
	void Serialize(Serializer &serializer) const;

	static unique_ptr<CreateInfo> Deserialize(Deserializer &deserializer);
	static unique_ptr<CreateInfo> Deserialize(Deserializer &deserializer, PlanDeserializationState &state);

	virtual unique_ptr<CreateInfo> Copy() const = 0;

	DUCKDB_API void CopyProperties(CreateInfo &other) const;
	//! Generates an alter statement from the create statement - used for OnCreateConflict::ALTER_ON_CONFLICT
	DUCKDB_API virtual unique_ptr<AlterInfo> GetAlterInfo() const;
};

} // namespace duckdb



namespace duckdb {

enum class SequenceInfo : uint8_t {
	// Sequence start
	SEQ_START,
	// Sequence increment
	SEQ_INC,
	// Sequence minimum value
	SEQ_MIN,
	// Sequence maximum value
	SEQ_MAX,
	// Sequence cycle option
	SEQ_CYCLE,
	// Sequence owner table
	SEQ_OWN
};

struct CreateSequenceInfo : public CreateInfo {
	CreateSequenceInfo();

	//! Sequence name to create
	string name;
	//! Usage count of the sequence
	uint64_t usage_count;
	//! The increment value
	int64_t increment;
	//! The minimum value of the sequence
	int64_t min_value;
	//! The maximum value of the sequence
	int64_t max_value;
	//! The start value of the sequence
	int64_t start_value;
	//! Whether or not the sequence cycles
	bool cycle;

public:
	unique_ptr<CreateInfo> Copy() const override;

public:
	DUCKDB_API static unique_ptr<CreateSequenceInfo> Deserialize(Deserializer &deserializer);

protected:
	void SerializeInternal(Serializer &) const override;
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/parsed_data/alter_table_info.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/parsed_data/alter_info.hpp
//
//
//===----------------------------------------------------------------------===//








namespace duckdb {

enum class AlterType : uint8_t {
	INVALID = 0,
	ALTER_TABLE = 1,
	ALTER_VIEW = 2,
	ALTER_SEQUENCE = 3,
	CHANGE_OWNERSHIP = 4,
	ALTER_SCALAR_FUNCTION = 5,
	ALTER_TABLE_FUNCTION = 6
};

struct AlterEntryData {
	AlterEntryData() {
	}
	AlterEntryData(string catalog_p, string schema_p, string name_p, OnEntryNotFound if_not_found)
	    : catalog(std::move(catalog_p)), schema(std::move(schema_p)), name(std::move(name_p)),
	      if_not_found(if_not_found) {
	}

	string catalog;
	string schema;
	string name;
	OnEntryNotFound if_not_found;
};

struct AlterInfo : public ParseInfo {
	AlterInfo(AlterType type, string catalog, string schema, string name, OnEntryNotFound if_not_found);
	virtual ~AlterInfo() override;

	AlterType type;
	//! if exists
	OnEntryNotFound if_not_found;
	//! Catalog name to alter
	string catalog;
	//! Schema name to alter
	string schema;
	//! Entry name to alter
	string name;
	//! Allow altering internal entries
	bool allow_internal;

public:
	virtual CatalogType GetCatalogType() const = 0;
	virtual unique_ptr<AlterInfo> Copy() const = 0;
	void Serialize(Serializer &serializer) const;
	virtual void Serialize(FieldWriter &writer) const = 0;
	static unique_ptr<AlterInfo> Deserialize(Deserializer &source);
	virtual string GetColumnName() const {
		return "";
	};

	AlterEntryData GetAlterEntryData() const;
};

} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/constraint.hpp
//
//
//===----------------------------------------------------------------------===//







namespace duckdb {

class Serializer;
class Deserializer;
class FieldWriter;
class FieldReader;

//===--------------------------------------------------------------------===//
// Constraint Types
//===--------------------------------------------------------------------===//
enum class ConstraintType : uint8_t {
	INVALID = 0,     // invalid constraint type
	NOT_NULL = 1,    // NOT NULL constraint
	CHECK = 2,       // CHECK constraint
	UNIQUE = 3,      // UNIQUE constraint
	FOREIGN_KEY = 4, // FOREIGN KEY constraint
};

enum class ForeignKeyType : uint8_t {
	FK_TYPE_PRIMARY_KEY_TABLE = 0,   // main table
	FK_TYPE_FOREIGN_KEY_TABLE = 1,   // referencing table
	FK_TYPE_SELF_REFERENCE_TABLE = 2 // self refrencing table
};

struct ForeignKeyInfo {
	ForeignKeyType type;
	string schema;
	//! if type is FK_TYPE_FOREIGN_KEY_TABLE, means main key table, if type is FK_TYPE_PRIMARY_KEY_TABLE, means foreign
	//! key table
	string table;
	//! The set of main key table's column's index
	vector<PhysicalIndex> pk_keys;
	//! The set of foreign key table's column's index
	vector<PhysicalIndex> fk_keys;
};

//! Constraint is the base class of any type of table constraint.
class Constraint {
public:
	DUCKDB_API explicit Constraint(ConstraintType type);
	DUCKDB_API virtual ~Constraint();

	ConstraintType type;

public:
	DUCKDB_API virtual string ToString() const = 0;
	DUCKDB_API void Print() const;

	DUCKDB_API virtual unique_ptr<Constraint> Copy() const = 0;
	//! Serializes a Constraint to a stand-alone binary blob
	DUCKDB_API void Serialize(Serializer &serializer) const;
	//! Serializes a Constraint to a stand-alone binary blob
	DUCKDB_API virtual void Serialize(FieldWriter &writer) const = 0;
	//! Deserializes a blob back into a Constraint
	DUCKDB_API static unique_ptr<Constraint> Deserialize(Deserializer &source);

public:
	template <class TARGET>
	TARGET &Cast() {
		if (type != TARGET::TYPE) {
			throw InternalException("Failed to cast constraint to type - constraint type mismatch");
		}
		return reinterpret_cast<TARGET &>(*this);
	}

	template <class TARGET>
	const TARGET &Cast() const {
		if (type != TARGET::TYPE) {
			throw InternalException("Failed to cast constraint to type - constraint type mismatch");
		}
		return reinterpret_cast<const TARGET &>(*this);
	}
};
} // namespace duckdb



namespace duckdb {

enum AlterForeignKeyType : uint8_t { AFT_ADD = 0, AFT_DELETE = 1 };

//===--------------------------------------------------------------------===//
// Change Ownership
//===--------------------------------------------------------------------===//
struct ChangeOwnershipInfo : public AlterInfo {
	ChangeOwnershipInfo(CatalogType entry_catalog_type, string entry_catalog, string entry_schema, string entry_name,
	                    string owner_schema, string owner_name, OnEntryNotFound if_not_found);

	// Catalog type refers to the entry type, since this struct is usually built from an
	// ALTER <TYPE> <schema>.<name> OWNED BY <owner_schema>.<owner_name> statement
	// here it is only possible to know the type of who is to be owned
	CatalogType entry_catalog_type;

	string owner_schema;
	string owner_name;

public:
	CatalogType GetCatalogType() const override;
	unique_ptr<AlterInfo> Copy() const override;
	void Serialize(FieldWriter &writer) const override;
};

//===--------------------------------------------------------------------===//
// Alter Table
//===--------------------------------------------------------------------===//
enum class AlterTableType : uint8_t {
	INVALID = 0,
	RENAME_COLUMN = 1,
	RENAME_TABLE = 2,
	ADD_COLUMN = 3,
	REMOVE_COLUMN = 4,
	ALTER_COLUMN_TYPE = 5,
	SET_DEFAULT = 6,
	FOREIGN_KEY_CONSTRAINT = 7,
	SET_NOT_NULL = 8,
	DROP_NOT_NULL = 9
};

struct AlterTableInfo : public AlterInfo {
	AlterTableInfo(AlterTableType type, AlterEntryData data);
	~AlterTableInfo() override;

	AlterTableType alter_table_type;

public:
	CatalogType GetCatalogType() const override;
	void Serialize(FieldWriter &writer) const override;
	virtual void SerializeAlterTable(FieldWriter &writer) const = 0;
	static unique_ptr<AlterInfo> Deserialize(FieldReader &reader);
};

//===--------------------------------------------------------------------===//
// RenameColumnInfo
//===--------------------------------------------------------------------===//
struct RenameColumnInfo : public AlterTableInfo {
	RenameColumnInfo(AlterEntryData data, string old_name_p, string new_name_p);
	~RenameColumnInfo() override;

	//! Column old name
	string old_name;
	//! Column new name
	string new_name;

public:
	unique_ptr<AlterInfo> Copy() const override;
	void SerializeAlterTable(FieldWriter &writer) const override;
	static unique_ptr<AlterInfo> Deserialize(FieldReader &reader, AlterEntryData data);
};

//===--------------------------------------------------------------------===//
// RenameTableInfo
//===--------------------------------------------------------------------===//
struct RenameTableInfo : public AlterTableInfo {
	RenameTableInfo(AlterEntryData data, string new_name);
	~RenameTableInfo() override;

	//! Relation new name
	string new_table_name;

public:
	unique_ptr<AlterInfo> Copy() const override;
	void SerializeAlterTable(FieldWriter &writer) const override;
	static unique_ptr<AlterInfo> Deserialize(FieldReader &reader, AlterEntryData data);
};

//===--------------------------------------------------------------------===//
// AddColumnInfo
//===--------------------------------------------------------------------===//
struct AddColumnInfo : public AlterTableInfo {
	AddColumnInfo(AlterEntryData data, ColumnDefinition new_column, bool if_column_not_exists);
	~AddColumnInfo() override;

	//! New column
	ColumnDefinition new_column;
	//! Whether or not an error should be thrown if the column exist
	bool if_column_not_exists;

public:
	unique_ptr<AlterInfo> Copy() const override;
	void SerializeAlterTable(FieldWriter &writer) const override;
	static unique_ptr<AlterInfo> Deserialize(FieldReader &reader, AlterEntryData data);
};

//===--------------------------------------------------------------------===//
// RemoveColumnInfo
//===--------------------------------------------------------------------===//
struct RemoveColumnInfo : public AlterTableInfo {
	RemoveColumnInfo(AlterEntryData data, string removed_column, bool if_column_exists, bool cascade);
	~RemoveColumnInfo() override;

	//! The column to remove
	string removed_column;
	//! Whether or not an error should be thrown if the column does not exist
	bool if_column_exists;
	//! Whether or not the column should be removed if a dependency conflict arises (used by GENERATED columns)
	bool cascade;

public:
	unique_ptr<AlterInfo> Copy() const override;
	void SerializeAlterTable(FieldWriter &writer) const override;
	static unique_ptr<AlterInfo> Deserialize(FieldReader &reader, AlterEntryData data);
	string GetColumnName() const override {
		return removed_column;
	};
};

//===--------------------------------------------------------------------===//
// ChangeColumnTypeInfo
//===--------------------------------------------------------------------===//
struct ChangeColumnTypeInfo : public AlterTableInfo {
	ChangeColumnTypeInfo(AlterEntryData data, string column_name, LogicalType target_type,
	                     unique_ptr<ParsedExpression> expression);
	~ChangeColumnTypeInfo() override;

	//! The column name to alter
	string column_name;
	//! The target type of the column
	LogicalType target_type;
	//! The expression used for data conversion
	unique_ptr<ParsedExpression> expression;

public:
	unique_ptr<AlterInfo> Copy() const override;
	void SerializeAlterTable(FieldWriter &writer) const override;
	static unique_ptr<AlterInfo> Deserialize(FieldReader &reader, AlterEntryData data);
	string GetColumnName() const override {
		return column_name;
	};
};

//===--------------------------------------------------------------------===//
// SetDefaultInfo
//===--------------------------------------------------------------------===//
struct SetDefaultInfo : public AlterTableInfo {
	SetDefaultInfo(AlterEntryData data, string column_name, unique_ptr<ParsedExpression> new_default);
	~SetDefaultInfo() override;

	//! The column name to alter
	string column_name;
	//! The expression used for data conversion
	unique_ptr<ParsedExpression> expression;

public:
	unique_ptr<AlterInfo> Copy() const override;
	void SerializeAlterTable(FieldWriter &writer) const override;
	static unique_ptr<AlterInfo> Deserialize(FieldReader &reader, AlterEntryData data);
};

//===--------------------------------------------------------------------===//
// AlterForeignKeyInfo
//===--------------------------------------------------------------------===//
struct AlterForeignKeyInfo : public AlterTableInfo {
	AlterForeignKeyInfo(AlterEntryData data, string fk_table, vector<string> pk_columns, vector<string> fk_columns,
	                    vector<PhysicalIndex> pk_keys, vector<PhysicalIndex> fk_keys, AlterForeignKeyType type);
	~AlterForeignKeyInfo() override;

	string fk_table;
	vector<string> pk_columns;
	vector<string> fk_columns;
	vector<PhysicalIndex> pk_keys;
	vector<PhysicalIndex> fk_keys;
	AlterForeignKeyType type;

public:
	unique_ptr<AlterInfo> Copy() const override;
	void SerializeAlterTable(FieldWriter &writer) const override;
	static unique_ptr<AlterInfo> Deserialize(FieldReader &reader, AlterEntryData data);
};

//===--------------------------------------------------------------------===//
// SetNotNullInfo
//===--------------------------------------------------------------------===//
struct SetNotNullInfo : public AlterTableInfo {
	SetNotNullInfo(AlterEntryData data, string column_name);
	~SetNotNullInfo() override;

	//! The column name to alter
	string column_name;

public:
	unique_ptr<AlterInfo> Copy() const override;
	void SerializeAlterTable(FieldWriter &writer) const override;
	static unique_ptr<AlterInfo> Deserialize(FieldReader &reader, AlterEntryData data);
};

//===--------------------------------------------------------------------===//
// DropNotNullInfo
//===--------------------------------------------------------------------===//
struct DropNotNullInfo : public AlterTableInfo {
	DropNotNullInfo(AlterEntryData data, string column_name);
	~DropNotNullInfo() override;

	//! The column name to alter
	string column_name;

public:
	unique_ptr<AlterInfo> Copy() const override;
	void SerializeAlterTable(FieldWriter &writer) const override;
	static unique_ptr<AlterInfo> Deserialize(FieldReader &reader, AlterEntryData data);
};

//===--------------------------------------------------------------------===//
// Alter View
//===--------------------------------------------------------------------===//
enum class AlterViewType : uint8_t { INVALID = 0, RENAME_VIEW = 1 };

struct AlterViewInfo : public AlterInfo {
	AlterViewInfo(AlterViewType type, AlterEntryData data);
	~AlterViewInfo() override;

	AlterViewType alter_view_type;

public:
	CatalogType GetCatalogType() const override;
	void Serialize(FieldWriter &writer) const override;
	virtual void SerializeAlterView(FieldWriter &writer) const = 0;
	static unique_ptr<AlterInfo> Deserialize(FieldReader &reader);
};

//===--------------------------------------------------------------------===//
// RenameViewInfo
//===--------------------------------------------------------------------===//
struct RenameViewInfo : public AlterViewInfo {
	RenameViewInfo(AlterEntryData data, string new_name);
	~RenameViewInfo() override;

	//! Relation new name
	string new_view_name;

public:
	unique_ptr<AlterInfo> Copy() const override;
	void SerializeAlterView(FieldWriter &writer) const override;
	static unique_ptr<AlterInfo> Deserialize(FieldReader &reader, AlterEntryData data);
};

} // namespace duckdb


namespace duckdb {
class Serializer;
class Deserializer;

struct SequenceValue {
	SequenceValue() : usage_count(0), counter(-1) {
	}
	SequenceValue(uint64_t usage_count, int64_t counter) : usage_count(usage_count), counter(counter) {
	}

	uint64_t usage_count;
	int64_t counter;
};

//! A sequence catalog entry
class SequenceCatalogEntry : public StandardEntry {
public:
	static constexpr const CatalogType Type = CatalogType::SEQUENCE_ENTRY;
	static constexpr const char *Name = "sequence";

public:
	//! Create a real TableCatalogEntry and initialize storage for it
	SequenceCatalogEntry(Catalog &catalog, SchemaCatalogEntry &schema, CreateSequenceInfo &info);

	//! Lock for getting a value on the sequence
	mutex lock;
	//! The amount of times the sequence has been used
	uint64_t usage_count;
	//! The sequence counter
	int64_t counter;
	//! The most recently returned value
	int64_t last_value;
	//! The increment value
	int64_t increment;
	//! The minimum value of the sequence
	int64_t start_value;
	//! The minimum value of the sequence
	int64_t min_value;
	//! The maximum value of the sequence
	int64_t max_value;
	//! Whether or not the sequence cycles
	bool cycle;

public:
	//! Serialize the meta information of the SequenceCatalogEntry a serializer
	virtual void Serialize(Serializer &serializer) const;
	//! Deserializes to a CreateTableInfo
	static unique_ptr<CreateSequenceInfo> Deserialize(Deserializer &source);

	string ToSQL() const override;

	CatalogEntry *AlterOwnership(ClientContext &context, AlterInfo *info);
};
} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/transaction/undo_buffer.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/undo_flags.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

enum class UndoFlags : uint32_t { // far too big but aligned (TM)
	EMPTY_ENTRY = 0,
	CATALOG_ENTRY = 1,
	INSERT_TUPLE = 2,
	DELETE_TUPLE = 3,
	UPDATE_TUPLE = 4
};

} // namespace duckdb



namespace duckdb {

class WriteAheadLog;

//! The undo buffer of a transaction is used to hold previous versions of tuples
//! that might be required in the future (because of rollbacks or previous
//! transactions accessing them)
class UndoBuffer {
public:
	struct IteratorState {
		ArenaChunk *current;
		data_ptr_t start;
		data_ptr_t end;
	};

public:
	UndoBuffer(ClientContext &context);

	//! Reserve space for an entry of the specified type and length in the undo
	//! buffer
	data_ptr_t CreateEntry(UndoFlags type, idx_t len);

	bool ChangesMade();
	idx_t EstimatedSize();

	//! Cleanup the undo buffer
	void Cleanup();
	//! Commit the changes made in the UndoBuffer: should be called on commit
	void Commit(UndoBuffer::IteratorState &iterator_state, optional_ptr<WriteAheadLog> log, transaction_t commit_id);
	//! Revert committed changes made in the UndoBuffer up until the currently committed state
	void RevertCommit(UndoBuffer::IteratorState &iterator_state, transaction_t transaction_id);
	//! Rollback the changes made in this UndoBuffer: should be called on
	//! rollback
	void Rollback() noexcept;

private:
	ClientContext &context;
	ArenaAllocator allocator;

private:
	template <class T>
	void IterateEntries(UndoBuffer::IteratorState &state, T &&callback);
	template <class T>
	void IterateEntries(UndoBuffer::IteratorState &state, UndoBuffer::IteratorState &end_state, T &&callback);
	template <class T>
	void ReverseIterateEntries(T &&callback);
};

} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/transaction/transaction_data.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {
class DuckTransaction;
class Transaction;

struct TransactionData {
	TransactionData(DuckTransaction &transaction_p);
	TransactionData(transaction_t transaction_id_p, transaction_t start_time_p);

	optional_ptr<DuckTransaction> transaction;
	transaction_t transaction_id;
	transaction_t start_time;
};

} // namespace duckdb


namespace duckdb {
class SequenceCatalogEntry;
class SchemaCatalogEntry;

class AttachedDatabase;
class ColumnData;
class ClientContext;
class CatalogEntry;
class DataTable;
class DatabaseInstance;
class LocalStorage;
class MetaTransaction;
class TransactionManager;
class WriteAheadLog;

class ChunkVectorInfo;

struct DeleteInfo;
struct UpdateInfo;

//! The transaction object holds information about a currently running or past
//! transaction
class Transaction {
public:
	DUCKDB_API Transaction(TransactionManager &manager, ClientContext &context);
	DUCKDB_API virtual ~Transaction();

	TransactionManager &manager;
	weak_ptr<ClientContext> context;
	//! The current active query for the transaction. Set to MAXIMUM_QUERY_ID if
	//! no query is active.
	atomic<transaction_t> active_query;

public:
	DUCKDB_API static Transaction &Get(ClientContext &context, AttachedDatabase &db);
	DUCKDB_API static Transaction &Get(ClientContext &context, Catalog &catalog);

	//! Whether or not the transaction has made any modifications to the database so far
	DUCKDB_API bool IsReadOnly();

	virtual bool IsDuckTransaction() const {
		return false;
	}

public:
	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}
	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
};

} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/catalog/similar_catalog_entry.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {
class SchemaCatalogEntry;

//! Return value of SimilarEntryInSchemas
struct SimilarCatalogEntry {
	//! The entry name. Empty if absent
	string name;
	//! The distance to the given name.
	idx_t distance = idx_t(-1);
	//! The schema of the entry.
	optional_ptr<SchemaCatalogEntry> schema;

	bool Found() const {
		return !name.empty();
	}

	DUCKDB_API string GetQualifiedName(bool qualify_catalog, bool qualify_schema) const;
};

} // namespace duckdb

#include <functional>
#include <memory>

namespace duckdb {
struct AlterInfo;

class ClientContext;
class DependencyList;
struct MappingValue;
struct EntryIndex;

class DuckCatalog;
class TableCatalogEntry;
class SequenceCatalogEntry;

typedef unordered_map<CatalogSet *, unique_lock<mutex>> set_lock_map_t;

struct EntryValue {
	EntryValue() {
		throw InternalException("EntryValue called without a catalog entry");
	}

	explicit EntryValue(unique_ptr<CatalogEntry> entry_p) : entry(std::move(entry_p)), reference_count(0) {
	}
	//! enable move constructors
	EntryValue(EntryValue &&other) noexcept {
		Swap(other);
	}
	EntryValue &operator=(EntryValue &&other) noexcept {
		Swap(other);
		return *this;
	}
	void Swap(EntryValue &other) {
		std::swap(entry, other.entry);
		idx_t count = reference_count;
		reference_count = other.reference_count.load();
		other.reference_count = count;
	}

	unique_ptr<CatalogEntry> entry;
	atomic<idx_t> reference_count;
};

//! The Catalog Set stores (key, value) map of a set of CatalogEntries
class CatalogSet {
	friend class DependencyManager;
	friend class EntryDropper;
	friend struct EntryIndex;

public:
	DUCKDB_API explicit CatalogSet(Catalog &catalog, unique_ptr<DefaultGenerator> defaults = nullptr);
	~CatalogSet();

	//! Create an entry in the catalog set. Returns whether or not it was
	//! successful.
	DUCKDB_API bool CreateEntry(CatalogTransaction transaction, const string &name, unique_ptr<CatalogEntry> value,
	                            DependencyList &dependencies);
	DUCKDB_API bool CreateEntry(ClientContext &context, const string &name, unique_ptr<CatalogEntry> value,
	                            DependencyList &dependencies);

	DUCKDB_API bool AlterEntry(CatalogTransaction transaction, const string &name, AlterInfo &alter_info);

	DUCKDB_API bool DropEntry(CatalogTransaction transaction, const string &name, bool cascade,
	                          bool allow_drop_internal = false);
	DUCKDB_API bool DropEntry(ClientContext &context, const string &name, bool cascade,
	                          bool allow_drop_internal = false);

	DUCKDB_API DuckCatalog &GetCatalog();

	bool AlterOwnership(CatalogTransaction transaction, ChangeOwnershipInfo &info);

	void CleanupEntry(CatalogEntry &catalog_entry);

	//! Returns the entry with the specified name
	DUCKDB_API optional_ptr<CatalogEntry> GetEntry(CatalogTransaction transaction, const string &name);
	DUCKDB_API optional_ptr<CatalogEntry> GetEntry(ClientContext &context, const string &name);

	//! Gets the entry that is most similar to the given name (i.e. smallest levenshtein distance), or empty string if
	//! none is found. The returned pair consists of the entry name and the distance (smaller means closer).
	SimilarCatalogEntry SimilarEntry(CatalogTransaction transaction, const string &name);

	//! Rollback <entry> to be the currently valid entry for a certain catalog
	//! entry
	void Undo(CatalogEntry &entry);

	//! Scan the catalog set, invoking the callback method for every committed entry
	DUCKDB_API void Scan(const std::function<void(CatalogEntry &)> &callback);
	//! Scan the catalog set, invoking the callback method for every entry
	DUCKDB_API void Scan(CatalogTransaction transaction, const std::function<void(CatalogEntry &)> &callback);
	DUCKDB_API void Scan(ClientContext &context, const std::function<void(CatalogEntry &)> &callback);

	template <class T>
	vector<reference<T>> GetEntries(CatalogTransaction transaction) {
		vector<reference<T>> result;
		Scan(transaction, [&](CatalogEntry &entry) { result.push_back(entry.Cast<T>()); });
		return result;
	}

	DUCKDB_API bool HasConflict(CatalogTransaction transaction, transaction_t timestamp);
	DUCKDB_API bool UseTimestamp(CatalogTransaction transaction, transaction_t timestamp);

	void UpdateTimestamp(CatalogEntry &entry, transaction_t timestamp);

	void Verify(Catalog &catalog);

private:
	//! Adjusts table dependencies on the event of an UNDO
	void AdjustTableDependencies(CatalogEntry &entry);
	//! Adjust one dependency
	void AdjustDependency(CatalogEntry &entry, TableCatalogEntry &table, ColumnDefinition &column, bool remove);
	//! Adjust User dependency
	void AdjustUserDependency(CatalogEntry &entry, ColumnDefinition &column, bool remove);
	//! Given a root entry, gets the entry valid for this transaction
	CatalogEntry &GetEntryForTransaction(CatalogTransaction transaction, CatalogEntry &current);
	CatalogEntry &GetCommittedEntry(CatalogEntry &current);
	optional_ptr<CatalogEntry> GetEntryInternal(CatalogTransaction transaction, const string &name,
	                                            EntryIndex *entry_index);
	optional_ptr<CatalogEntry> GetEntryInternal(CatalogTransaction transaction, EntryIndex &entry_index);
	//! Drops an entry from the catalog set; must hold the catalog_lock to safely call this
	void DropEntryInternal(CatalogTransaction transaction, EntryIndex entry_index, CatalogEntry &entry, bool cascade);
	optional_ptr<CatalogEntry> CreateEntryInternal(CatalogTransaction transaction, unique_ptr<CatalogEntry> entry);
	optional_ptr<MappingValue> GetMapping(CatalogTransaction transaction, const string &name, bool get_latest = false);
	void PutMapping(CatalogTransaction transaction, const string &name, EntryIndex entry_index);
	void DeleteMapping(CatalogTransaction transaction, const string &name);
	void DropEntryDependencies(CatalogTransaction transaction, EntryIndex &entry_index, CatalogEntry &entry,
	                           bool cascade);

	//! Create all default entries
	void CreateDefaultEntries(CatalogTransaction transaction, unique_lock<mutex> &lock);
	//! Attempt to create a default entry with the specified name. Returns the entry if successful, nullptr otherwise.
	optional_ptr<CatalogEntry> CreateDefaultEntry(CatalogTransaction transaction, const string &name,
	                                              unique_lock<mutex> &lock);

	EntryIndex PutEntry(idx_t entry_index, unique_ptr<CatalogEntry> entry);
	void PutEntry(EntryIndex index, unique_ptr<CatalogEntry> entry);

private:
	DuckCatalog &catalog;
	//! The catalog lock is used to make changes to the data
	mutex catalog_lock;
	//! The set of catalog entries
	unordered_map<idx_t, EntryValue> entries;
	//! Mapping of string to catalog entry
	case_insensitive_map_t<unique_ptr<MappingValue>> mapping;
	//! The current catalog entry index
	idx_t current_entry = 0;
	//! The generator used to generate default internal entries
	unique_ptr<DefaultGenerator> defaults;
};
} // namespace duckdb



namespace duckdb {
class ClientContext;

class StandardEntry;
class TableCatalogEntry;
class TableFunctionCatalogEntry;
class SequenceCatalogEntry;
class Serializer;
class Deserializer;

enum class OnCreateConflict : uint8_t;

struct AlterTableInfo;
struct CreateIndexInfo;
struct CreateFunctionInfo;
struct CreateCollationInfo;
struct CreateViewInfo;
struct BoundCreateTableInfo;
struct CreatePragmaFunctionInfo;
struct CreateSequenceInfo;
struct CreateSchemaInfo;
struct CreateTableFunctionInfo;
struct CreateCopyFunctionInfo;
struct CreateTypeInfo;

struct DropInfo;

//! A schema in the catalog
class SchemaCatalogEntry : public InCatalogEntry {
public:
	static constexpr const CatalogType Type = CatalogType::SCHEMA_ENTRY;
	static constexpr const char *Name = "schema";

public:
	SchemaCatalogEntry(Catalog &catalog, string name, bool is_internal);

public:
	//! Scan the specified catalog set, invoking the callback method for every entry
	virtual void Scan(ClientContext &context, CatalogType type,
	                  const std::function<void(CatalogEntry &)> &callback) = 0;
	//! Scan the specified catalog set, invoking the callback method for every committed entry
	virtual void Scan(CatalogType type, const std::function<void(CatalogEntry &)> &callback) = 0;

	//! Serialize the meta information of the SchemaCatalogEntry a serializer
	virtual void Serialize(Serializer &serializer) const;
	//! Deserializes to a CreateSchemaInfo
	static unique_ptr<CreateSchemaInfo> Deserialize(Deserializer &source);

	string ToSQL() const override;

	//! Creates an index with the given name in the schema
	virtual optional_ptr<CatalogEntry> CreateIndex(ClientContext &context, CreateIndexInfo &info,
	                                               TableCatalogEntry &table) = 0;
	//! Create a scalar or aggregate function within the given schema
	virtual optional_ptr<CatalogEntry> CreateFunction(CatalogTransaction transaction, CreateFunctionInfo &info) = 0;
	//! Creates a table with the given name in the schema
	virtual optional_ptr<CatalogEntry> CreateTable(CatalogTransaction transaction, BoundCreateTableInfo &info) = 0;
	//! Creates a view with the given name in the schema
	virtual optional_ptr<CatalogEntry> CreateView(CatalogTransaction transaction, CreateViewInfo &info) = 0;
	//! Creates a sequence with the given name in the schema
	virtual optional_ptr<CatalogEntry> CreateSequence(CatalogTransaction transaction, CreateSequenceInfo &info) = 0;
	//! Create a table function within the given schema
	virtual optional_ptr<CatalogEntry> CreateTableFunction(CatalogTransaction transaction,
	                                                       CreateTableFunctionInfo &info) = 0;
	//! Create a copy function within the given schema
	virtual optional_ptr<CatalogEntry> CreateCopyFunction(CatalogTransaction transaction,
	                                                      CreateCopyFunctionInfo &info) = 0;
	//! Create a pragma function within the given schema
	virtual optional_ptr<CatalogEntry> CreatePragmaFunction(CatalogTransaction transaction,
	                                                        CreatePragmaFunctionInfo &info) = 0;
	//! Create a collation within the given schema
	virtual optional_ptr<CatalogEntry> CreateCollation(CatalogTransaction transaction, CreateCollationInfo &info) = 0;
	//! Create a enum within the given schema
	virtual optional_ptr<CatalogEntry> CreateType(CatalogTransaction transaction, CreateTypeInfo &info) = 0;

	DUCKDB_API virtual optional_ptr<CatalogEntry> GetEntry(CatalogTransaction transaction, CatalogType type,
	                                                       const string &name) = 0;
	DUCKDB_API virtual SimilarCatalogEntry GetSimilarEntry(CatalogTransaction transaction, CatalogType type,
	                                                       const string &name);

	//! Drops an entry from the schema
	virtual void DropEntry(ClientContext &context, DropInfo &info) = 0;

	//! Alters a catalog entry
	virtual void Alter(ClientContext &context, AlterInfo &info) = 0;

	CatalogTransaction GetCatalogTransaction(ClientContext &context);
};
} // namespace duckdb



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/deque.hpp
//
//
//===----------------------------------------------------------------------===//



#include <deque>

namespace duckdb {
using std::deque;
}





//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/main/stream_query_result.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {

class ClientContext;
class ClientContextLock;
class Executor;
class MaterializedQueryResult;
class PreparedStatementData;

class StreamQueryResult : public QueryResult {
	friend class ClientContext;

public:
	static constexpr const QueryResultType TYPE = QueryResultType::STREAM_RESULT;

public:
	//! Create a successful StreamQueryResult. StreamQueryResults should always be successful initially (it makes no
	//! sense to stream an error).
	DUCKDB_API StreamQueryResult(StatementType statement_type, StatementProperties properties,
	                             shared_ptr<ClientContext> context, vector<LogicalType> types, vector<string> names);
	DUCKDB_API ~StreamQueryResult() override;

public:
	//! Fetches a DataChunk from the query result.
	DUCKDB_API unique_ptr<DataChunk> FetchRaw() override;
	//! Converts the QueryResult to a string
	DUCKDB_API string ToString() override;
	//! Materializes the query result and turns it into a materialized query result
	DUCKDB_API unique_ptr<MaterializedQueryResult> Materialize();

	DUCKDB_API bool IsOpen();

	//! Closes the StreamQueryResult
	DUCKDB_API void Close();

	//! The client context this StreamQueryResult belongs to
	shared_ptr<ClientContext> context;

private:
	unique_ptr<ClientContextLock> LockContext();
	void CheckExecutableInternal(ClientContextLock &lock);
	bool IsOpenInternal(ClientContextLock &lock);
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/main/table_description.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

struct TableDescription {
	//! The schema of the table
	string schema;
	//! The table name of the table
	string table;
	//! The columns of the table
	vector<ColumnDefinition> columns;
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/transaction/transaction_context.hpp
//
//
//===----------------------------------------------------------------------===//







namespace duckdb {

class ClientContext;
class MetaTransaction;
class Transaction;
class TransactionManager;

//! The transaction context keeps track of all the information relating to the
//! current transaction
class TransactionContext {
public:
	TransactionContext(ClientContext &context);
	~TransactionContext();

	MetaTransaction &ActiveTransaction() {
		if (!current_transaction) {
			throw InternalException("TransactionContext::ActiveTransaction called without active transaction");
		}
		return *current_transaction;
	}

	bool HasActiveTransaction() {
		return !!current_transaction;
	}

	void BeginTransaction();
	void Commit();
	void Rollback();
	void ClearTransaction();

	void SetAutoCommit(bool value);
	bool IsAutoCommit() {
		return auto_commit;
	}

	idx_t GetActiveQuery();
	void ResetActiveQuery();
	void SetActiveQuery(transaction_t query_number);

private:
	ClientContext &context;
	bool auto_commit;

	unique_ptr<MetaTransaction> current_transaction;

	TransactionContext(const TransactionContext &) = delete;
};

} // namespace duckdb



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/main/client_config.hpp
//
//
//===----------------------------------------------------------------------===//





//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/output_type.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

enum class ExplainOutputType : uint8_t { ALL = 0, OPTIMIZED_ONLY = 1, PHYSICAL_ONLY = 2 };

} // namespace duckdb



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/progress_bar/progress_bar.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//
//                         DuckDB
//
// duckdb.h
//
//
//===----------------------------------------------------------------------===//



// duplicate of duckdb/main/winapi.hpp
#ifndef DUCKDB_API
#ifdef _WIN32
#if defined(DUCKDB_BUILD_LIBRARY) && !defined(DUCKDB_BUILD_LOADABLE_EXTENSION)
#define DUCKDB_API __declspec(dllexport)
#else
#define DUCKDB_API __declspec(dllimport)
#endif
#else
#define DUCKDB_API
#endif
#endif

// duplicate of duckdb/main/winapi.hpp
#ifndef DUCKDB_EXTENSION_API
#ifdef _WIN32
#ifdef DUCKDB_BUILD_LOADABLE_EXTENSION
#define DUCKDB_EXTENSION_API __declspec(dllexport)
#else
#define DUCKDB_EXTENSION_API
#endif
#else
#define DUCKDB_EXTENSION_API __attribute__((visibility("default")))
#endif
#endif

// API versions
// if no explicit API version is defined, the latest API version is used
// Note that using older API versions (i.e. not using DUCKDB_API_LATEST) is deprecated.
// These will not be supported long-term, and will be removed in future versions.
#ifndef DUCKDB_API_0_3_1
#define DUCKDB_API_0_3_1 1
#endif
#ifndef DUCKDB_API_0_3_2
#define DUCKDB_API_0_3_2 2
#endif
#ifndef DUCKDB_API_LATEST
#define DUCKDB_API_LATEST DUCKDB_API_0_3_2
#endif

#ifndef DUCKDB_API_VERSION
#define DUCKDB_API_VERSION DUCKDB_API_LATEST
#endif

#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>

#ifdef __cplusplus
extern "C" {
#endif

//===--------------------------------------------------------------------===//
// Type Information
//===--------------------------------------------------------------------===//
typedef uint64_t idx_t;

typedef enum DUCKDB_TYPE {
	DUCKDB_TYPE_INVALID = 0,
	// bool
	DUCKDB_TYPE_BOOLEAN,
	// int8_t
	DUCKDB_TYPE_TINYINT,
	// int16_t
	DUCKDB_TYPE_SMALLINT,
	// int32_t
	DUCKDB_TYPE_INTEGER,
	// int64_t
	DUCKDB_TYPE_BIGINT,
	// uint8_t
	DUCKDB_TYPE_UTINYINT,
	// uint16_t
	DUCKDB_TYPE_USMALLINT,
	// uint32_t
	DUCKDB_TYPE_UINTEGER,
	// uint64_t
	DUCKDB_TYPE_UBIGINT,
	// float
	DUCKDB_TYPE_FLOAT,
	// double
	DUCKDB_TYPE_DOUBLE,
	// duckdb_timestamp, in microseconds
	DUCKDB_TYPE_TIMESTAMP,
	// duckdb_date
	DUCKDB_TYPE_DATE,
	// duckdb_time
	DUCKDB_TYPE_TIME,
	// duckdb_interval
	DUCKDB_TYPE_INTERVAL,
	// duckdb_hugeint
	DUCKDB_TYPE_HUGEINT,
	// const char*
	DUCKDB_TYPE_VARCHAR,
	// duckdb_blob
	DUCKDB_TYPE_BLOB,
	// decimal
	DUCKDB_TYPE_DECIMAL,
	// duckdb_timestamp, in seconds
	DUCKDB_TYPE_TIMESTAMP_S,
	// duckdb_timestamp, in milliseconds
	DUCKDB_TYPE_TIMESTAMP_MS,
	// duckdb_timestamp, in nanoseconds
	DUCKDB_TYPE_TIMESTAMP_NS,
	// enum type, only useful as logical type
	DUCKDB_TYPE_ENUM,
	// list type, only useful as logical type
	DUCKDB_TYPE_LIST,
	// struct type, only useful as logical type
	DUCKDB_TYPE_STRUCT,
	// map type, only useful as logical type
	DUCKDB_TYPE_MAP,
	// duckdb_hugeint
	DUCKDB_TYPE_UUID,
	// union type, only useful as logical type
	DUCKDB_TYPE_UNION,
	// duckdb_bit
	DUCKDB_TYPE_BIT,
} duckdb_type;

//! Days are stored as days since 1970-01-01
//! Use the duckdb_from_date/duckdb_to_date function to extract individual information
typedef struct {
	int32_t days;
} duckdb_date;

typedef struct {
	int32_t year;
	int8_t month;
	int8_t day;
} duckdb_date_struct;

//! Time is stored as microseconds since 00:00:00
//! Use the duckdb_from_time/duckdb_to_time function to extract individual information
typedef struct {
	int64_t micros;
} duckdb_time;

typedef struct {
	int8_t hour;
	int8_t min;
	int8_t sec;
	int32_t micros;
} duckdb_time_struct;

//! Timestamps are stored as microseconds since 1970-01-01
//! Use the duckdb_from_timestamp/duckdb_to_timestamp function to extract individual information
typedef struct {
	int64_t micros;
} duckdb_timestamp;

typedef struct {
	duckdb_date_struct date;
	duckdb_time_struct time;
} duckdb_timestamp_struct;

typedef struct {
	int32_t months;
	int32_t days;
	int64_t micros;
} duckdb_interval;

//! Hugeints are composed in a (lower, upper) component
//! The value of the hugeint is upper * 2^64 + lower
//! For easy usage, the functions duckdb_hugeint_to_double/duckdb_double_to_hugeint are recommended
typedef struct {
	uint64_t lower;
	int64_t upper;
} duckdb_hugeint;

typedef struct {
	uint8_t width;
	uint8_t scale;

	duckdb_hugeint value;
} duckdb_decimal;

typedef struct {
	char *data;
	idx_t size;
} duckdb_string;

/*
    The internal data representation of a VARCHAR/BLOB column
*/
typedef struct {
	union {
		struct {
			uint32_t length;
			char prefix[4];
			char *ptr;
		} pointer;
		struct {
			uint32_t length;
			char inlined[12];
		} inlined;
	} value;
} duckdb_string_t;

typedef struct {
	void *data;
	idx_t size;
} duckdb_blob;

typedef struct {
	uint64_t offset;
	uint64_t length;
} duckdb_list_entry;

typedef struct {
#if DUCKDB_API_VERSION < DUCKDB_API_0_3_2
	void *data;
	bool *nullmask;
	duckdb_type type;
	char *name;
#else
	// deprecated, use duckdb_column_data
	void *__deprecated_data;
	// deprecated, use duckdb_nullmask_data
	bool *__deprecated_nullmask;
	// deprecated, use duckdb_column_type
	duckdb_type __deprecated_type;
	// deprecated, use duckdb_column_name
	char *__deprecated_name;
#endif
	void *internal_data;
} duckdb_column;

typedef struct {
#if DUCKDB_API_VERSION < DUCKDB_API_0_3_2
	idx_t column_count;
	idx_t row_count;
	idx_t rows_changed;
	duckdb_column *columns;
	char *error_message;
#else
	// deprecated, use duckdb_column_count
	idx_t __deprecated_column_count;
	// deprecated, use duckdb_row_count
	idx_t __deprecated_row_count;
	// deprecated, use duckdb_rows_changed
	idx_t __deprecated_rows_changed;
	// deprecated, use duckdb_column_ family of functions
	duckdb_column *__deprecated_columns;
	// deprecated, use duckdb_result_error
	char *__deprecated_error_message;
#endif
	void *internal_data;
} duckdb_result;

typedef struct _duckdb_database {
	void *__db;
} * duckdb_database;
typedef struct _duckdb_connection {
	void *__conn;
} * duckdb_connection;
typedef struct _duckdb_prepared_statement {
	void *__prep;
} * duckdb_prepared_statement;
typedef struct _duckdb_extracted_statements {
	void *__extrac;
} * duckdb_extracted_statements;
typedef struct _duckdb_pending_result {
	void *__pend;
} * duckdb_pending_result;
typedef struct _duckdb_appender {
	void *__appn;
} * duckdb_appender;
typedef struct _duckdb_arrow {
	void *__arrw;
} * duckdb_arrow;
typedef struct _duckdb_config {
	void *__cnfg;
} * duckdb_config;
typedef struct _duckdb_arrow_schema {
	void *__arrs;
} * duckdb_arrow_schema;
typedef struct _duckdb_arrow_array {
	void *__arra;
} * duckdb_arrow_array;
typedef struct _duckdb_logical_type {
	void *__lglt;
} * duckdb_logical_type;
typedef struct _duckdb_data_chunk {
	void *__dtck;
} * duckdb_data_chunk;
typedef struct _duckdb_vector {
	void *__vctr;
} * duckdb_vector;
typedef struct _duckdb_value {
	void *__val;
} * duckdb_value;

typedef enum { DuckDBSuccess = 0, DuckDBError = 1 } duckdb_state;
typedef enum {
	DUCKDB_PENDING_RESULT_READY = 0,
	DUCKDB_PENDING_RESULT_NOT_READY = 1,
	DUCKDB_PENDING_ERROR = 2
} duckdb_pending_state;

//===--------------------------------------------------------------------===//
// Open/Connect
//===--------------------------------------------------------------------===//

/*!
Creates a new database or opens an existing database file stored at the the given path.
If no path is given a new in-memory database is created instead.
The instantiated database should be closed with 'duckdb_close'

* path: Path to the database file on disk, or `nullptr` or `:memory:` to open an in-memory database.
* out_database: The result database object.
* returns: `DuckDBSuccess` on success or `DuckDBError` on failure.
*/
DUCKDB_API duckdb_state duckdb_open(const char *path, duckdb_database *out_database);

/*!
Extended version of duckdb_open. Creates a new database or opens an existing database file stored at the the given path.

* path: Path to the database file on disk, or `nullptr` or `:memory:` to open an in-memory database.
* out_database: The result database object.
* config: (Optional) configuration used to start up the database system.
* out_error: If set and the function returns DuckDBError, this will contain the reason why the start-up failed.
Note that the error must be freed using `duckdb_free`.
* returns: `DuckDBSuccess` on success or `DuckDBError` on failure.
*/
DUCKDB_API duckdb_state duckdb_open_ext(const char *path, duckdb_database *out_database, duckdb_config config,
                                        char **out_error);

/*!
Closes the specified database and de-allocates all memory allocated for that database.
This should be called after you are done with any database allocated through `duckdb_open`.
Note that failing to call `duckdb_close` (in case of e.g. a program crash) will not cause data corruption.
Still it is recommended to always correctly close a database object after you are done with it.

* database: The database object to shut down.
*/
DUCKDB_API void duckdb_close(duckdb_database *database);

/*!
Opens a connection to a database. Connections are required to query the database, and store transactional state
associated with the connection.
The instantiated connection should be closed using 'duckdb_disconnect'

* database: The database file to connect to.
* out_connection: The result connection object.
* returns: `DuckDBSuccess` on success or `DuckDBError` on failure.
*/
DUCKDB_API duckdb_state duckdb_connect(duckdb_database database, duckdb_connection *out_connection);

/*!
Closes the specified connection and de-allocates all memory allocated for that connection.

* connection: The connection to close.
*/
DUCKDB_API void duckdb_disconnect(duckdb_connection *connection);

/*!
Returns the version of the linked DuckDB, with a version postfix for dev versions

Usually used for developing C extensions that must return this for a compatibility check.
*/
DUCKDB_API const char *duckdb_library_version();

//===--------------------------------------------------------------------===//
// Configuration
//===--------------------------------------------------------------------===//
/*!
Initializes an empty configuration object that can be used to provide start-up options for the DuckDB instance
through `duckdb_open_ext`.

This will always succeed unless there is a malloc failure.

* out_config: The result configuration object.
* returns: `DuckDBSuccess` on success or `DuckDBError` on failure.
*/
DUCKDB_API duckdb_state duckdb_create_config(duckdb_config *out_config);

/*!
This returns the total amount of configuration options available for usage with `duckdb_get_config_flag`.

This should not be called in a loop as it internally loops over all the options.

* returns: The amount of config options available.
*/
DUCKDB_API size_t duckdb_config_count();

/*!
Obtains a human-readable name and description of a specific configuration option. This can be used to e.g.
display configuration options. This will succeed unless `index` is out of range (i.e. `>= duckdb_config_count`).

The result name or description MUST NOT be freed.

* index: The index of the configuration option (between 0 and `duckdb_config_count`)
* out_name: A name of the configuration flag.
* out_description: A description of the configuration flag.
* returns: `DuckDBSuccess` on success or `DuckDBError` on failure.
*/
DUCKDB_API duckdb_state duckdb_get_config_flag(size_t index, const char **out_name, const char **out_description);

/*!
Sets the specified option for the specified configuration. The configuration option is indicated by name.
To obtain a list of config options, see `duckdb_get_config_flag`.

In the source code, configuration options are defined in `config.cpp`.

This can fail if either the name is invalid, or if the value provided for the option is invalid.

* duckdb_config: The configuration object to set the option on.
* name: The name of the configuration flag to set.
* option: The value to set the configuration flag to.
* returns: `DuckDBSuccess` on success or `DuckDBError` on failure.
*/
DUCKDB_API duckdb_state duckdb_set_config(duckdb_config config, const char *name, const char *option);

/*!
Destroys the specified configuration option and de-allocates all memory allocated for the object.

* config: The configuration object to destroy.
*/
DUCKDB_API void duckdb_destroy_config(duckdb_config *config);

//===--------------------------------------------------------------------===//
// Query Execution
//===--------------------------------------------------------------------===//
/*!
Executes a SQL query within a connection and stores the full (materialized) result in the out_result pointer.
If the query fails to execute, DuckDBError is returned and the error message can be retrieved by calling
`duckdb_result_error`.

Note that after running `duckdb_query`, `duckdb_destroy_result` must be called on the result object even if the
query fails, otherwise the error stored within the result will not be freed correctly.

* connection: The connection to perform the query in.
* query: The SQL query to run.
* out_result: The query result.
* returns: `DuckDBSuccess` on success or `DuckDBError` on failure.
*/
DUCKDB_API duckdb_state duckdb_query(duckdb_connection connection, const char *query, duckdb_result *out_result);

/*!
Closes the result and de-allocates all memory allocated for that connection.

* result: The result to destroy.
*/
DUCKDB_API void duckdb_destroy_result(duckdb_result *result);

/*!
Returns the column name of the specified column. The result should not need be freed; the column names will
automatically be destroyed when the result is destroyed.

Returns `NULL` if the column is out of range.

* result: The result object to fetch the column name from.
* col: The column index.
* returns: The column name of the specified column.
*/
DUCKDB_API const char *duckdb_column_name(duckdb_result *result, idx_t col);

/*!
Returns the column type of the specified column.

Returns `DUCKDB_TYPE_INVALID` if the column is out of range.

* result: The result object to fetch the column type from.
* col: The column index.
* returns: The column type of the specified column.
*/
DUCKDB_API duckdb_type duckdb_column_type(duckdb_result *result, idx_t col);

/*!
Returns the logical column type of the specified column.

The return type of this call should be destroyed with `duckdb_destroy_logical_type`.

Returns `NULL` if the column is out of range.

* result: The result object to fetch the column type from.
* col: The column index.
* returns: The logical column type of the specified column.
*/
DUCKDB_API duckdb_logical_type duckdb_column_logical_type(duckdb_result *result, idx_t col);

/*!
Returns the number of columns present in a the result object.

* result: The result object.
* returns: The number of columns present in the result object.
*/
DUCKDB_API idx_t duckdb_column_count(duckdb_result *result);

/*!
Returns the number of rows present in a the result object.

* result: The result object.
* returns: The number of rows present in the result object.
*/
DUCKDB_API idx_t duckdb_row_count(duckdb_result *result);

/*!
Returns the number of rows changed by the query stored in the result. This is relevant only for INSERT/UPDATE/DELETE
queries. For other queries the rows_changed will be 0.

* result: The result object.
* returns: The number of rows changed.
*/
DUCKDB_API idx_t duckdb_rows_changed(duckdb_result *result);

/*!
**DEPRECATED**: Prefer using `duckdb_result_get_chunk` instead.

Returns the data of a specific column of a result in columnar format.

The function returns a dense array which contains the result data. The exact type stored in the array depends on the
corresponding duckdb_type (as provided by `duckdb_column_type`). For the exact type by which the data should be
accessed, see the comments in [the types section](types) or the `DUCKDB_TYPE` enum.

For example, for a column of type `DUCKDB_TYPE_INTEGER`, rows can be accessed in the following manner:
```c
int32_t *data = (int32_t *) duckdb_column_data(&result, 0);
printf("Data for row %d: %d\n", row, data[row]);
```

* result: The result object to fetch the column data from.
* col: The column index.
* returns: The column data of the specified column.
*/
DUCKDB_API void *duckdb_column_data(duckdb_result *result, idx_t col);

/*!
**DEPRECATED**: Prefer using `duckdb_result_get_chunk` instead.

Returns the nullmask of a specific column of a result in columnar format. The nullmask indicates for every row
whether or not the corresponding row is `NULL`. If a row is `NULL`, the values present in the array provided
by `duckdb_column_data` are undefined.

```c
int32_t *data = (int32_t *) duckdb_column_data(&result, 0);
bool *nullmask = duckdb_nullmask_data(&result, 0);
if (nullmask[row]) {
    printf("Data for row %d: NULL\n", row);
} else {
    printf("Data for row %d: %d\n", row, data[row]);
}
```

* result: The result object to fetch the nullmask from.
* col: The column index.
* returns: The nullmask of the specified column.
*/
DUCKDB_API bool *duckdb_nullmask_data(duckdb_result *result, idx_t col);

/*!
Returns the error message contained within the result. The error is only set if `duckdb_query` returns `DuckDBError`.

The result of this function must not be freed. It will be cleaned up when `duckdb_destroy_result` is called.

* result: The result object to fetch the error from.
* returns: The error of the result.
*/
DUCKDB_API const char *duckdb_result_error(duckdb_result *result);

//===--------------------------------------------------------------------===//
// Result Functions
//===--------------------------------------------------------------------===//

/*!
Fetches a data chunk from the duckdb_result. This function should be called repeatedly until the result is exhausted.

The result must be destroyed with `duckdb_destroy_data_chunk`.

This function supersedes all `duckdb_value` functions, as well as the `duckdb_column_data` and `duckdb_nullmask_data`
functions. It results in significantly better performance, and should be preferred in newer code-bases.

If this function is used, none of the other result functions can be used and vice versa (i.e. this function cannot be
mixed with the legacy result functions).

Use `duckdb_result_chunk_count` to figure out how many chunks there are in the result.

* result: The result object to fetch the data chunk from.
* chunk_index: The chunk index to fetch from.
* returns: The resulting data chunk. Returns `NULL` if the chunk index is out of bounds.
*/
DUCKDB_API duckdb_data_chunk duckdb_result_get_chunk(duckdb_result result, idx_t chunk_index);

/*!
Checks if the type of the internal result is StreamQueryResult.

* result: The result object to check.
* returns: Whether or not the result object is of the type StreamQueryResult
*/
DUCKDB_API bool duckdb_result_is_streaming(duckdb_result result);

/*!
Returns the number of data chunks present in the result.

* result: The result object
* returns: Number of data chunks present in the result.
*/
DUCKDB_API idx_t duckdb_result_chunk_count(duckdb_result result);

// Safe fetch functions
// These functions will perform conversions if necessary.
// On failure (e.g. if conversion cannot be performed or if the value is NULL) a default value is returned.
// Note that these functions are slow since they perform bounds checking and conversion
// For fast access of values prefer using `duckdb_result_get_chunk`

/*!
 * returns: The boolean value at the specified location, or false if the value cannot be converted.
 */
DUCKDB_API bool duckdb_value_boolean(duckdb_result *result, idx_t col, idx_t row);

/*!
 * returns: The int8_t value at the specified location, or 0 if the value cannot be converted.
 */
DUCKDB_API int8_t duckdb_value_int8(duckdb_result *result, idx_t col, idx_t row);

/*!
 * returns: The int16_t value at the specified location, or 0 if the value cannot be converted.
 */
DUCKDB_API int16_t duckdb_value_int16(duckdb_result *result, idx_t col, idx_t row);

/*!
 * returns: The int32_t value at the specified location, or 0 if the value cannot be converted.
 */
DUCKDB_API int32_t duckdb_value_int32(duckdb_result *result, idx_t col, idx_t row);

/*!
 * returns: The int64_t value at the specified location, or 0 if the value cannot be converted.
 */
DUCKDB_API int64_t duckdb_value_int64(duckdb_result *result, idx_t col, idx_t row);

/*!
 * returns: The duckdb_hugeint value at the specified location, or 0 if the value cannot be converted.
 */
DUCKDB_API duckdb_hugeint duckdb_value_hugeint(duckdb_result *result, idx_t col, idx_t row);

/*!
 * returns: The duckdb_decimal value at the specified location, or 0 if the value cannot be converted.
 */
DUCKDB_API duckdb_decimal duckdb_value_decimal(duckdb_result *result, idx_t col, idx_t row);

/*!
 * returns: The uint8_t value at the specified location, or 0 if the value cannot be converted.
 */
DUCKDB_API uint8_t duckdb_value_uint8(duckdb_result *result, idx_t col, idx_t row);

/*!
 * returns: The uint16_t value at the specified location, or 0 if the value cannot be converted.
 */
DUCKDB_API uint16_t duckdb_value_uint16(duckdb_result *result, idx_t col, idx_t row);

/*!
 * returns: The uint32_t value at the specified location, or 0 if the value cannot be converted.
 */
DUCKDB_API uint32_t duckdb_value_uint32(duckdb_result *result, idx_t col, idx_t row);

/*!
 * returns: The uint64_t value at the specified location, or 0 if the value cannot be converted.
 */
DUCKDB_API uint64_t duckdb_value_uint64(duckdb_result *result, idx_t col, idx_t row);

/*!
 * returns: The float value at the specified location, or 0 if the value cannot be converted.
 */
DUCKDB_API float duckdb_value_float(duckdb_result *result, idx_t col, idx_t row);

/*!
 * returns: The double value at the specified location, or 0 if the value cannot be converted.
 */
DUCKDB_API double duckdb_value_double(duckdb_result *result, idx_t col, idx_t row);

/*!
 * returns: The duckdb_date value at the specified location, or 0 if the value cannot be converted.
 */
DUCKDB_API duckdb_date duckdb_value_date(duckdb_result *result, idx_t col, idx_t row);

/*!
 * returns: The duckdb_time value at the specified location, or 0 if the value cannot be converted.
 */
DUCKDB_API duckdb_time duckdb_value_time(duckdb_result *result, idx_t col, idx_t row);

/*!
 * returns: The duckdb_timestamp value at the specified location, or 0 if the value cannot be converted.
 */
DUCKDB_API duckdb_timestamp duckdb_value_timestamp(duckdb_result *result, idx_t col, idx_t row);

/*!
 * returns: The duckdb_interval value at the specified location, or 0 if the value cannot be converted.
 */
DUCKDB_API duckdb_interval duckdb_value_interval(duckdb_result *result, idx_t col, idx_t row);

/*!
* DEPRECATED: use duckdb_value_string instead. This function does not work correctly if the string contains null bytes.
* returns: The text value at the specified location as a null-terminated string, or nullptr if the value cannot be
converted. The result must be freed with `duckdb_free`.
*/
DUCKDB_API char *duckdb_value_varchar(duckdb_result *result, idx_t col, idx_t row);

/*!s
* returns: The string value at the specified location.
The result must be freed with `duckdb_free`.
*/
DUCKDB_API duckdb_string duckdb_value_string(duckdb_result *result, idx_t col, idx_t row);

/*!
* DEPRECATED: use duckdb_value_string_internal instead. This function does not work correctly if the string contains
null bytes.
* returns: The char* value at the specified location. ONLY works on VARCHAR columns and does not auto-cast.
If the column is NOT a VARCHAR column this function will return NULL.

The result must NOT be freed.
*/
DUCKDB_API char *duckdb_value_varchar_internal(duckdb_result *result, idx_t col, idx_t row);

/*!
* DEPRECATED: use duckdb_value_string_internal instead. This function does not work correctly if the string contains
null bytes.
* returns: The char* value at the specified location. ONLY works on VARCHAR columns and does not auto-cast.
If the column is NOT a VARCHAR column this function will return NULL.

The result must NOT be freed.
*/
DUCKDB_API duckdb_string duckdb_value_string_internal(duckdb_result *result, idx_t col, idx_t row);

/*!
* returns: The duckdb_blob value at the specified location. Returns a blob with blob.data set to nullptr if the
value cannot be converted. The resulting "blob.data" must be freed with `duckdb_free.`
*/
DUCKDB_API duckdb_blob duckdb_value_blob(duckdb_result *result, idx_t col, idx_t row);

/*!
 * returns: Returns true if the value at the specified index is NULL, and false otherwise.
 */
DUCKDB_API bool duckdb_value_is_null(duckdb_result *result, idx_t col, idx_t row);

//===--------------------------------------------------------------------===//
// Helpers
//===--------------------------------------------------------------------===//
/*!
Allocate `size` bytes of memory using the duckdb internal malloc function. Any memory allocated in this manner
should be freed using `duckdb_free`.

* size: The number of bytes to allocate.
* returns: A pointer to the allocated memory region.
*/
DUCKDB_API void *duckdb_malloc(size_t size);

/*!
Free a value returned from `duckdb_malloc`, `duckdb_value_varchar` or `duckdb_value_blob`.

* ptr: The memory region to de-allocate.
*/
DUCKDB_API void duckdb_free(void *ptr);

/*!
The internal vector size used by DuckDB.
This is the amount of tuples that will fit into a data chunk created by `duckdb_create_data_chunk`.

* returns: The vector size.
*/
DUCKDB_API idx_t duckdb_vector_size();

/*!
Whether or not the duckdb_string_t value is inlined.
This means that the data of the string does not have a separate allocation.

*/
DUCKDB_API bool duckdb_string_is_inlined(duckdb_string_t string);

//===--------------------------------------------------------------------===//
// Date/Time/Timestamp Helpers
//===--------------------------------------------------------------------===//
/*!
Decompose a `duckdb_date` object into year, month and date (stored as `duckdb_date_struct`).

* date: The date object, as obtained from a `DUCKDB_TYPE_DATE` column.
* returns: The `duckdb_date_struct` with the decomposed elements.
*/
DUCKDB_API duckdb_date_struct duckdb_from_date(duckdb_date date);

/*!
Re-compose a `duckdb_date` from year, month and date (`duckdb_date_struct`).

* date: The year, month and date stored in a `duckdb_date_struct`.
* returns: The `duckdb_date` element.
*/
DUCKDB_API duckdb_date duckdb_to_date(duckdb_date_struct date);

/*!
Decompose a `duckdb_time` object into hour, minute, second and microsecond (stored as `duckdb_time_struct`).

* time: The time object, as obtained from a `DUCKDB_TYPE_TIME` column.
* returns: The `duckdb_time_struct` with the decomposed elements.
*/
DUCKDB_API duckdb_time_struct duckdb_from_time(duckdb_time time);

/*!
Re-compose a `duckdb_time` from hour, minute, second and microsecond (`duckdb_time_struct`).

* time: The hour, minute, second and microsecond in a `duckdb_time_struct`.
* returns: The `duckdb_time` element.
*/
DUCKDB_API duckdb_time duckdb_to_time(duckdb_time_struct time);

/*!
Decompose a `duckdb_timestamp` object into a `duckdb_timestamp_struct`.

* ts: The ts object, as obtained from a `DUCKDB_TYPE_TIMESTAMP` column.
* returns: The `duckdb_timestamp_struct` with the decomposed elements.
*/
DUCKDB_API duckdb_timestamp_struct duckdb_from_timestamp(duckdb_timestamp ts);

/*!
Re-compose a `duckdb_timestamp` from a duckdb_timestamp_struct.

* ts: The de-composed elements in a `duckdb_timestamp_struct`.
* returns: The `duckdb_timestamp` element.
*/
DUCKDB_API duckdb_timestamp duckdb_to_timestamp(duckdb_timestamp_struct ts);

//===--------------------------------------------------------------------===//
// Hugeint Helpers
//===--------------------------------------------------------------------===//
/*!
Converts a duckdb_hugeint object (as obtained from a `DUCKDB_TYPE_HUGEINT` column) into a double.

* val: The hugeint value.
* returns: The converted `double` element.
*/
DUCKDB_API double duckdb_hugeint_to_double(duckdb_hugeint val);

/*!
Converts a double value to a duckdb_hugeint object.

If the conversion fails because the double value is too big the result will be 0.

* val: The double value.
* returns: The converted `duckdb_hugeint` element.
*/
DUCKDB_API duckdb_hugeint duckdb_double_to_hugeint(double val);

/*!
Converts a double value to a duckdb_decimal object.

If the conversion fails because the double value is too big, or the width/scale are invalid the result will be 0.

* val: The double value.
* returns: The converted `duckdb_decimal` element.
*/
DUCKDB_API duckdb_decimal duckdb_double_to_decimal(double val, uint8_t width, uint8_t scale);

//===--------------------------------------------------------------------===//
// Decimal Helpers
//===--------------------------------------------------------------------===//
/*!
Converts a duckdb_decimal object (as obtained from a `DUCKDB_TYPE_DECIMAL` column) into a double.

* val: The decimal value.
* returns: The converted `double` element.
*/
DUCKDB_API double duckdb_decimal_to_double(duckdb_decimal val);

//===--------------------------------------------------------------------===//
// Prepared Statements
//===--------------------------------------------------------------------===//
// A prepared statement is a parameterized query that allows you to bind parameters to it.
// * This is useful to easily supply parameters to functions and avoid SQL injection attacks.
// * This is useful to speed up queries that you will execute several times with different parameters.
// Because the query will only be parsed, bound, optimized and planned once during the prepare stage,
// rather than once per execution.
// For example:
//   SELECT * FROM tbl WHERE id=?
// Or a query with multiple parameters:
//   SELECT * FROM tbl WHERE id=$1 OR name=$2

/*!
Create a prepared statement object from a query.

Note that after calling `duckdb_prepare`, the prepared statement should always be destroyed using
`duckdb_destroy_prepare`, even if the prepare fails.

If the prepare fails, `duckdb_prepare_error` can be called to obtain the reason why the prepare failed.

* connection: The connection object
* query: The SQL query to prepare
* out_prepared_statement: The resulting prepared statement object
* returns: `DuckDBSuccess` on success or `DuckDBError` on failure.
*/
DUCKDB_API duckdb_state duckdb_prepare(duckdb_connection connection, const char *query,
                                       duckdb_prepared_statement *out_prepared_statement);

/*!
Closes the prepared statement and de-allocates all memory allocated for the statement.

* prepared_statement: The prepared statement to destroy.
*/
DUCKDB_API void duckdb_destroy_prepare(duckdb_prepared_statement *prepared_statement);

/*!
Returns the error message associated with the given prepared statement.
If the prepared statement has no error message, this returns `nullptr` instead.

The error message should not be freed. It will be de-allocated when `duckdb_destroy_prepare` is called.

* prepared_statement: The prepared statement to obtain the error from.
* returns: The error message, or `nullptr` if there is none.
*/
DUCKDB_API const char *duckdb_prepare_error(duckdb_prepared_statement prepared_statement);

/*!
Returns the number of parameters that can be provided to the given prepared statement.

Returns 0 if the query was not successfully prepared.

* prepared_statement: The prepared statement to obtain the number of parameters for.
*/
DUCKDB_API idx_t duckdb_nparams(duckdb_prepared_statement prepared_statement);

/*!
Returns the parameter type for the parameter at the given index.

Returns `DUCKDB_TYPE_INVALID` if the parameter index is out of range or the statement was not successfully prepared.

* prepared_statement: The prepared statement.
* param_idx: The parameter index.
* returns: The parameter type
*/
DUCKDB_API duckdb_type duckdb_param_type(duckdb_prepared_statement prepared_statement, idx_t param_idx);

/*!
Clear the params bind to the prepared statement.
*/
DUCKDB_API duckdb_state duckdb_clear_bindings(duckdb_prepared_statement prepared_statement);

/*!
Binds a bool value to the prepared statement at the specified index.
*/
DUCKDB_API duckdb_state duckdb_bind_boolean(duckdb_prepared_statement prepared_statement, idx_t param_idx, bool val);

/*!
Binds an int8_t value to the prepared statement at the specified index.
*/
DUCKDB_API duckdb_state duckdb_bind_int8(duckdb_prepared_statement prepared_statement, idx_t param_idx, int8_t val);

/*!
Binds an int16_t value to the prepared statement at the specified index.
*/
DUCKDB_API duckdb_state duckdb_bind_int16(duckdb_prepared_statement prepared_statement, idx_t param_idx, int16_t val);

/*!
Binds an int32_t value to the prepared statement at the specified index.
*/
DUCKDB_API duckdb_state duckdb_bind_int32(duckdb_prepared_statement prepared_statement, idx_t param_idx, int32_t val);

/*!
Binds an int64_t value to the prepared statement at the specified index.
*/
DUCKDB_API duckdb_state duckdb_bind_int64(duckdb_prepared_statement prepared_statement, idx_t param_idx, int64_t val);

/*!
Binds an duckdb_hugeint value to the prepared statement at the specified index.
*/
DUCKDB_API duckdb_state duckdb_bind_hugeint(duckdb_prepared_statement prepared_statement, idx_t param_idx,
                                            duckdb_hugeint val);
/*!
Binds a duckdb_decimal value to the prepared statement at the specified index.
*/
DUCKDB_API duckdb_state duckdb_bind_decimal(duckdb_prepared_statement prepared_statement, idx_t param_idx,
                                            duckdb_decimal val);

/*!
Binds an uint8_t value to the prepared statement at the specified index.
*/
DUCKDB_API duckdb_state duckdb_bind_uint8(duckdb_prepared_statement prepared_statement, idx_t param_idx, uint8_t val);

/*!
Binds an uint16_t value to the prepared statement at the specified index.
*/
DUCKDB_API duckdb_state duckdb_bind_uint16(duckdb_prepared_statement prepared_statement, idx_t param_idx, uint16_t val);

/*!
Binds an uint32_t value to the prepared statement at the specified index.
*/
DUCKDB_API duckdb_state duckdb_bind_uint32(duckdb_prepared_statement prepared_statement, idx_t param_idx, uint32_t val);

/*!
Binds an uint64_t value to the prepared statement at the specified index.
*/
DUCKDB_API duckdb_state duckdb_bind_uint64(duckdb_prepared_statement prepared_statement, idx_t param_idx, uint64_t val);

/*!
Binds an float value to the prepared statement at the specified index.
*/
DUCKDB_API duckdb_state duckdb_bind_float(duckdb_prepared_statement prepared_statement, idx_t param_idx, float val);

/*!
Binds an double value to the prepared statement at the specified index.
*/
DUCKDB_API duckdb_state duckdb_bind_double(duckdb_prepared_statement prepared_statement, idx_t param_idx, double val);

/*!
Binds a duckdb_date value to the prepared statement at the specified index.
*/
DUCKDB_API duckdb_state duckdb_bind_date(duckdb_prepared_statement prepared_statement, idx_t param_idx,
                                         duckdb_date val);

/*!
Binds a duckdb_time value to the prepared statement at the specified index.
*/
DUCKDB_API duckdb_state duckdb_bind_time(duckdb_prepared_statement prepared_statement, idx_t param_idx,
                                         duckdb_time val);

/*!
Binds a duckdb_timestamp value to the prepared statement at the specified index.
*/
DUCKDB_API duckdb_state duckdb_bind_timestamp(duckdb_prepared_statement prepared_statement, idx_t param_idx,
                                              duckdb_timestamp val);

/*!
Binds a duckdb_interval value to the prepared statement at the specified index.
*/
DUCKDB_API duckdb_state duckdb_bind_interval(duckdb_prepared_statement prepared_statement, idx_t param_idx,
                                             duckdb_interval val);

/*!
Binds a null-terminated varchar value to the prepared statement at the specified index.
*/
DUCKDB_API duckdb_state duckdb_bind_varchar(duckdb_prepared_statement prepared_statement, idx_t param_idx,
                                            const char *val);

/*!
Binds a varchar value to the prepared statement at the specified index.
*/
DUCKDB_API duckdb_state duckdb_bind_varchar_length(duckdb_prepared_statement prepared_statement, idx_t param_idx,
                                                   const char *val, idx_t length);

/*!
Binds a blob value to the prepared statement at the specified index.
*/
DUCKDB_API duckdb_state duckdb_bind_blob(duckdb_prepared_statement prepared_statement, idx_t param_idx,
                                         const void *data, idx_t length);

/*!
Binds a NULL value to the prepared statement at the specified index.
*/
DUCKDB_API duckdb_state duckdb_bind_null(duckdb_prepared_statement prepared_statement, idx_t param_idx);

/*!
Executes the prepared statement with the given bound parameters, and returns a materialized query result.

This method can be called multiple times for each prepared statement, and the parameters can be modified
between calls to this function.

* prepared_statement: The prepared statement to execute.
* out_result: The query result.
* returns: `DuckDBSuccess` on success or `DuckDBError` on failure.
*/
DUCKDB_API duckdb_state duckdb_execute_prepared(duckdb_prepared_statement prepared_statement,
                                                duckdb_result *out_result);

/*!
Executes the prepared statement with the given bound parameters, and returns an arrow query result.

* prepared_statement: The prepared statement to execute.
* out_result: The query result.
* returns: `DuckDBSuccess` on success or `DuckDBError` on failure.
*/
DUCKDB_API duckdb_state duckdb_execute_prepared_arrow(duckdb_prepared_statement prepared_statement,
                                                      duckdb_arrow *out_result);

//===--------------------------------------------------------------------===//
// Extract Statements
//===--------------------------------------------------------------------===//
// A query string can be extracted into multiple SQL statements. Each statement can be prepared and executed separately.

/*!
Extract all statements from a query.
Note that after calling `duckdb_extract_statements`, the extracted statements should always be destroyed using
`duckdb_destroy_extracted`, even if no statements were extracted.
If the extract fails, `duckdb_extract_statements_error` can be called to obtain the reason why the extract failed.
* connection: The connection object
* query: The SQL query to extract
* out_extracted_statements: The resulting extracted statements object
* returns: The number of extracted statements or 0 on failure.
*/
DUCKDB_API idx_t duckdb_extract_statements(duckdb_connection connection, const char *query,
                                           duckdb_extracted_statements *out_extracted_statements);

/*!
Prepare an extracted statement.
Note that after calling `duckdb_prepare_extracted_statement`, the prepared statement should always be destroyed using
`duckdb_destroy_prepare`, even if the prepare fails.
If the prepare fails, `duckdb_prepare_error` can be called to obtain the reason why the prepare failed.
* connection: The connection object
* extracted_statements: The extracted statements object
* index: The index of the extracted statement to prepare
* out_prepared_statement: The resulting prepared statement object
* returns: `DuckDBSuccess` on success or `DuckDBError` on failure.
*/
DUCKDB_API duckdb_state duckdb_prepare_extracted_statement(duckdb_connection connection,
                                                           duckdb_extracted_statements extracted_statements,
                                                           idx_t index,
                                                           duckdb_prepared_statement *out_prepared_statement);
/*!
Returns the error message contained within the extracted statements.
The result of this function must not be freed. It will be cleaned up when `duckdb_destroy_extracted` is called.
* result: The extracted statements to fetch the error from.
* returns: The error of the extracted statements.
*/
DUCKDB_API const char *duckdb_extract_statements_error(duckdb_extracted_statements extracted_statements);

/*!
De-allocates all memory allocated for the extracted statements.
* extracted_statements: The extracted statements to destroy.
*/
DUCKDB_API void duckdb_destroy_extracted(duckdb_extracted_statements *extracted_statements);

//===--------------------------------------------------------------------===//
// Pending Result Interface
//===--------------------------------------------------------------------===//
/*!
Executes the prepared statement with the given bound parameters, and returns a pending result.
The pending result represents an intermediate structure for a query that is not yet fully executed.
The pending result can be used to incrementally execute a query, returning control to the client between tasks.

Note that after calling `duckdb_pending_prepared`, the pending result should always be destroyed using
`duckdb_destroy_pending`, even if this function returns DuckDBError.

* prepared_statement: The prepared statement to execute.
* out_result: The pending query result.
* returns: `DuckDBSuccess` on success or `DuckDBError` on failure.
*/
DUCKDB_API duckdb_state duckdb_pending_prepared(duckdb_prepared_statement prepared_statement,
                                                duckdb_pending_result *out_result);

/*!
Executes the prepared statement with the given bound parameters, and returns a pending result.
This pending result will create a streaming duckdb_result when executed.
The pending result represents an intermediate structure for a query that is not yet fully executed.

Note that after calling `duckdb_pending_prepared_streaming`, the pending result should always be destroyed using
`duckdb_destroy_pending`, even if this function returns DuckDBError.

* prepared_statement: The prepared statement to execute.
* out_result: The pending query result.
* returns: `DuckDBSuccess` on success or `DuckDBError` on failure.
*/
DUCKDB_API duckdb_state duckdb_pending_prepared_streaming(duckdb_prepared_statement prepared_statement,
                                                          duckdb_pending_result *out_result);

/*!
Closes the pending result and de-allocates all memory allocated for the result.

* pending_result: The pending result to destroy.
*/
DUCKDB_API void duckdb_destroy_pending(duckdb_pending_result *pending_result);

/*!
Returns the error message contained within the pending result.

The result of this function must not be freed. It will be cleaned up when `duckdb_destroy_pending` is called.

* result: The pending result to fetch the error from.
* returns: The error of the pending result.
*/
DUCKDB_API const char *duckdb_pending_error(duckdb_pending_result pending_result);

/*!
Executes a single task within the query, returning whether or not the query is ready.

If this returns DUCKDB_PENDING_RESULT_READY, the duckdb_execute_pending function can be called to obtain the result.
If this returns DUCKDB_PENDING_RESULT_NOT_READY, the duckdb_pending_execute_task function should be called again.
If this returns DUCKDB_PENDING_ERROR, an error occurred during execution.

The error message can be obtained by calling duckdb_pending_error on the pending_result.

* pending_result: The pending result to execute a task within..
* returns: The state of the pending result after the execution.
*/
DUCKDB_API duckdb_pending_state duckdb_pending_execute_task(duckdb_pending_result pending_result);

/*!
Fully execute a pending query result, returning the final query result.

If duckdb_pending_execute_task has been called until DUCKDB_PENDING_RESULT_READY was returned, this will return fast.
Otherwise, all remaining tasks must be executed first.

* pending_result: The pending result to execute.
* out_result: The result object.
* returns: `DuckDBSuccess` on success or `DuckDBError` on failure.
*/
DUCKDB_API duckdb_state duckdb_execute_pending(duckdb_pending_result pending_result, duckdb_result *out_result);

//===--------------------------------------------------------------------===//
// Value Interface
//===--------------------------------------------------------------------===//
/*!
Destroys the value and de-allocates all memory allocated for that type.

* value: The value to destroy.
*/
DUCKDB_API void duckdb_destroy_value(duckdb_value *value);

/*!
Creates a value from a null-terminated string

* value: The null-terminated string
* returns: The value. This must be destroyed with `duckdb_destroy_value`.
*/
DUCKDB_API duckdb_value duckdb_create_varchar(const char *text);

/*!
Creates a value from a string

* value: The text
* length: The length of the text
* returns: The value. This must be destroyed with `duckdb_destroy_value`.
*/
DUCKDB_API duckdb_value duckdb_create_varchar_length(const char *text, idx_t length);

/*!
Creates a value from an int64

* value: The bigint value
* returns: The value. This must be destroyed with `duckdb_destroy_value`.
*/
DUCKDB_API duckdb_value duckdb_create_int64(int64_t val);

/*!
Obtains a string representation of the given value.
The result must be destroyed with `duckdb_free`.

* value: The value
* returns: The string value. This must be destroyed with `duckdb_free`.
*/
DUCKDB_API char *duckdb_get_varchar(duckdb_value value);

/*!
Obtains an int64 of the given value.

* value: The value
* returns: The int64 value, or 0 if no conversion is possible
*/
DUCKDB_API int64_t duckdb_get_int64(duckdb_value value);

//===--------------------------------------------------------------------===//
// Logical Type Interface
//===--------------------------------------------------------------------===//

/*!
Creates a `duckdb_logical_type` from a standard primitive type.
The resulting type should be destroyed with `duckdb_destroy_logical_type`.

This should not be used with `DUCKDB_TYPE_DECIMAL`.

* type: The primitive type to create.
* returns: The logical type.
*/
DUCKDB_API duckdb_logical_type duckdb_create_logical_type(duckdb_type type);

/*!
Creates a list type from its child type.
The resulting type should be destroyed with `duckdb_destroy_logical_type`.

* type: The child type of list type to create.
* returns: The logical type.
*/
DUCKDB_API duckdb_logical_type duckdb_create_list_type(duckdb_logical_type type);

/*!
Creates a map type from its key type and value type.
The resulting type should be destroyed with `duckdb_destroy_logical_type`.

* type: The key type and value type of map type to create.
* returns: The logical type.
*/
DUCKDB_API duckdb_logical_type duckdb_create_map_type(duckdb_logical_type key_type, duckdb_logical_type value_type);

/*!
Creates a UNION type from the passed types array
The resulting type should be destroyed with `duckdb_destroy_logical_type`.

* types: The array of types that the union should consist of.
* type_amount: The size of the types array.
* returns: The logical type.
*/
DUCKDB_API duckdb_logical_type duckdb_create_union_type(duckdb_logical_type member_types, const char **member_names,
                                                        idx_t member_count);

/*!
Creates a `duckdb_logical_type` of type decimal with the specified width and scale
The resulting type should be destroyed with `duckdb_destroy_logical_type`.

* width: The width of the decimal type
* scale: The scale of the decimal type
* returns: The logical type.
*/
DUCKDB_API duckdb_logical_type duckdb_create_decimal_type(uint8_t width, uint8_t scale);

/*!
Retrieves the type class of a `duckdb_logical_type`.

* type: The logical type object
* returns: The type id
*/
DUCKDB_API duckdb_type duckdb_get_type_id(duckdb_logical_type type);

/*!
Retrieves the width of a decimal type.

* type: The logical type object
* returns: The width of the decimal type
*/
DUCKDB_API uint8_t duckdb_decimal_width(duckdb_logical_type type);

/*!
Retrieves the scale of a decimal type.

* type: The logical type object
* returns: The scale of the decimal type
*/
DUCKDB_API uint8_t duckdb_decimal_scale(duckdb_logical_type type);

/*!
Retrieves the internal storage type of a decimal type.

* type: The logical type object
* returns: The internal type of the decimal type
*/
DUCKDB_API duckdb_type duckdb_decimal_internal_type(duckdb_logical_type type);

/*!
Retrieves the internal storage type of an enum type.

* type: The logical type object
* returns: The internal type of the enum type
*/
DUCKDB_API duckdb_type duckdb_enum_internal_type(duckdb_logical_type type);

/*!
Retrieves the dictionary size of the enum type

* type: The logical type object
* returns: The dictionary size of the enum type
*/
DUCKDB_API uint32_t duckdb_enum_dictionary_size(duckdb_logical_type type);

/*!
Retrieves the dictionary value at the specified position from the enum.

The result must be freed with `duckdb_free`

* type: The logical type object
* index: The index in the dictionary
* returns: The string value of the enum type. Must be freed with `duckdb_free`.
*/
DUCKDB_API char *duckdb_enum_dictionary_value(duckdb_logical_type type, idx_t index);

/*!
Retrieves the child type of the given list type.

The result must be freed with `duckdb_destroy_logical_type`

* type: The logical type object
* returns: The child type of the list type. Must be destroyed with `duckdb_destroy_logical_type`.
*/
DUCKDB_API duckdb_logical_type duckdb_list_type_child_type(duckdb_logical_type type);

/*!
Retrieves the key type of the given map type.

The result must be freed with `duckdb_destroy_logical_type`

* type: The logical type object
* returns: The key type of the map type. Must be destroyed with `duckdb_destroy_logical_type`.
*/
DUCKDB_API duckdb_logical_type duckdb_map_type_key_type(duckdb_logical_type type);

/*!
Retrieves the value type of the given map type.

The result must be freed with `duckdb_destroy_logical_type`

* type: The logical type object
* returns: The value type of the map type. Must be destroyed with `duckdb_destroy_logical_type`.
*/
DUCKDB_API duckdb_logical_type duckdb_map_type_value_type(duckdb_logical_type type);

/*!
Returns the number of children of a struct type.

* type: The logical type object
* returns: The number of children of a struct type.
*/
DUCKDB_API idx_t duckdb_struct_type_child_count(duckdb_logical_type type);

/*!
Retrieves the name of the struct child.

The result must be freed with `duckdb_free`

* type: The logical type object
* index: The child index
* returns: The name of the struct type. Must be freed with `duckdb_free`.
*/
DUCKDB_API char *duckdb_struct_type_child_name(duckdb_logical_type type, idx_t index);

/*!
Retrieves the child type of the given struct type at the specified index.

The result must be freed with `duckdb_destroy_logical_type`

* type: The logical type object
* index: The child index
* returns: The child type of the struct type. Must be destroyed with `duckdb_destroy_logical_type`.
*/
DUCKDB_API duckdb_logical_type duckdb_struct_type_child_type(duckdb_logical_type type, idx_t index);

/*!
Returns the number of members that the union type has.

* type: The logical type (union) object
* returns: The number of members of a union type.
*/
DUCKDB_API idx_t duckdb_union_type_member_count(duckdb_logical_type type);

/*!
Retrieves the name of the union member.

The result must be freed with `duckdb_free`

* type: The logical type object
* index: The child index
* returns: The name of the union member. Must be freed with `duckdb_free`.
*/
DUCKDB_API char *duckdb_union_type_member_name(duckdb_logical_type type, idx_t index);

/*!
Retrieves the child type of the given union member at the specified index.

The result must be freed with `duckdb_destroy_logical_type`

* type: The logical type object
* index: The child index
* returns: The child type of the union member. Must be destroyed with `duckdb_destroy_logical_type`.
*/
DUCKDB_API duckdb_logical_type duckdb_union_type_member_type(duckdb_logical_type type, idx_t index);

/*!
Destroys the logical type and de-allocates all memory allocated for that type.

* type: The logical type to destroy.
*/
DUCKDB_API void duckdb_destroy_logical_type(duckdb_logical_type *type);

//===--------------------------------------------------------------------===//
// Data Chunk Interface
//===--------------------------------------------------------------------===//
/*!
Creates an empty DataChunk with the specified set of types.

* types: An array of types of the data chunk.
* column_count: The number of columns.
* returns: The data chunk.
*/
DUCKDB_API duckdb_data_chunk duckdb_create_data_chunk(duckdb_logical_type *types, idx_t column_count);

/*!
Destroys the data chunk and de-allocates all memory allocated for that chunk.

* chunk: The data chunk to destroy.
*/
DUCKDB_API void duckdb_destroy_data_chunk(duckdb_data_chunk *chunk);

/*!
Resets a data chunk, clearing the validity masks and setting the cardinality of the data chunk to 0.

* chunk: The data chunk to reset.
*/
DUCKDB_API void duckdb_data_chunk_reset(duckdb_data_chunk chunk);

/*!
Retrieves the number of columns in a data chunk.

* chunk: The data chunk to get the data from
* returns: The number of columns in the data chunk
*/
DUCKDB_API idx_t duckdb_data_chunk_get_column_count(duckdb_data_chunk chunk);

/*!
Retrieves the vector at the specified column index in the data chunk.

The pointer to the vector is valid for as long as the chunk is alive.
It does NOT need to be destroyed.

* chunk: The data chunk to get the data from
* returns: The vector
*/
DUCKDB_API duckdb_vector duckdb_data_chunk_get_vector(duckdb_data_chunk chunk, idx_t col_idx);

/*!
Retrieves the current number of tuples in a data chunk.

* chunk: The data chunk to get the data from
* returns: The number of tuples in the data chunk
*/
DUCKDB_API idx_t duckdb_data_chunk_get_size(duckdb_data_chunk chunk);

/*!
Sets the current number of tuples in a data chunk.

* chunk: The data chunk to set the size in
* size: The number of tuples in the data chunk
*/
DUCKDB_API void duckdb_data_chunk_set_size(duckdb_data_chunk chunk, idx_t size);

//===--------------------------------------------------------------------===//
// Vector Interface
//===--------------------------------------------------------------------===//
/*!
Retrieves the column type of the specified vector.

The result must be destroyed with `duckdb_destroy_logical_type`.

* vector: The vector get the data from
* returns: The type of the vector
*/
DUCKDB_API duckdb_logical_type duckdb_vector_get_column_type(duckdb_vector vector);

/*!
Retrieves the data pointer of the vector.

The data pointer can be used to read or write values from the vector.
How to read or write values depends on the type of the vector.

* vector: The vector to get the data from
* returns: The data pointer
*/
DUCKDB_API void *duckdb_vector_get_data(duckdb_vector vector);

/*!
Retrieves the validity mask pointer of the specified vector.

If all values are valid, this function MIGHT return NULL!

The validity mask is a bitset that signifies null-ness within the data chunk.
It is a series of uint64_t values, where each uint64_t value contains validity for 64 tuples.
The bit is set to 1 if the value is valid (i.e. not NULL) or 0 if the value is invalid (i.e. NULL).

Validity of a specific value can be obtained like this:

idx_t entry_idx = row_idx / 64;
idx_t idx_in_entry = row_idx % 64;
bool is_valid = validity_mask[entry_idx] & (1 << idx_in_entry);

Alternatively, the (slower) duckdb_validity_row_is_valid function can be used.

* vector: The vector to get the data from
* returns: The pointer to the validity mask, or NULL if no validity mask is present
*/
DUCKDB_API uint64_t *duckdb_vector_get_validity(duckdb_vector vector);

/*!
Ensures the validity mask is writable by allocating it.

After this function is called, `duckdb_vector_get_validity` will ALWAYS return non-NULL.
This allows null values to be written to the vector, regardless of whether a validity mask was present before.

* vector: The vector to alter
*/
DUCKDB_API void duckdb_vector_ensure_validity_writable(duckdb_vector vector);

/*!
Assigns a string element in the vector at the specified location.

* vector: The vector to alter
* index: The row position in the vector to assign the string to
* str: The null-terminated string
*/
DUCKDB_API void duckdb_vector_assign_string_element(duckdb_vector vector, idx_t index, const char *str);

/*!
Assigns a string element in the vector at the specified location.

* vector: The vector to alter
* index: The row position in the vector to assign the string to
* str: The string
* str_len: The length of the string (in bytes)
*/
DUCKDB_API void duckdb_vector_assign_string_element_len(duckdb_vector vector, idx_t index, const char *str,
                                                        idx_t str_len);

/*!
Retrieves the child vector of a list vector.

The resulting vector is valid as long as the parent vector is valid.

* vector: The vector
* returns: The child vector
*/
DUCKDB_API duckdb_vector duckdb_list_vector_get_child(duckdb_vector vector);

/*!
Returns the size of the child vector of the list

* vector: The vector
* returns: The size of the child list
*/
DUCKDB_API idx_t duckdb_list_vector_get_size(duckdb_vector vector);

/*!
Sets the total size of the underlying child-vector of a list vector.

* vector: The list vector.
* size: The size of the child list.
* returns: The duckdb state. Returns DuckDBError if the vector is nullptr.
*/
DUCKDB_API duckdb_state duckdb_list_vector_set_size(duckdb_vector vector, idx_t size);

/*!
Sets the total capacity of the underlying child-vector of a list.

* vector: The list vector.
* required_capacity: the total capacity to reserve.
* return: The duckdb state. Returns DuckDBError if the vector is nullptr.
*/
DUCKDB_API duckdb_state duckdb_list_vector_reserve(duckdb_vector vector, idx_t required_capacity);

/*!
Retrieves the child vector of a struct vector.

The resulting vector is valid as long as the parent vector is valid.

* vector: The vector
* index: The child index
* returns: The child vector
*/
DUCKDB_API duckdb_vector duckdb_struct_vector_get_child(duckdb_vector vector, idx_t index);

//===--------------------------------------------------------------------===//
// Validity Mask Functions
//===--------------------------------------------------------------------===//
/*!
Returns whether or not a row is valid (i.e. not NULL) in the given validity mask.

* validity: The validity mask, as obtained through `duckdb_data_chunk_get_validity`
* row: The row index
* returns: true if the row is valid, false otherwise
*/
DUCKDB_API bool duckdb_validity_row_is_valid(uint64_t *validity, idx_t row);

/*!
In a validity mask, sets a specific row to either valid or invalid.

Note that `duckdb_data_chunk_ensure_validity_writable` should be called before calling `duckdb_data_chunk_get_validity`,
to ensure that there is a validity mask to write to.

* validity: The validity mask, as obtained through `duckdb_data_chunk_get_validity`.
* row: The row index
* valid: Whether or not to set the row to valid, or invalid
*/
DUCKDB_API void duckdb_validity_set_row_validity(uint64_t *validity, idx_t row, bool valid);

/*!
In a validity mask, sets a specific row to invalid.

Equivalent to `duckdb_validity_set_row_validity` with valid set to false.

* validity: The validity mask
* row: The row index
*/
DUCKDB_API void duckdb_validity_set_row_invalid(uint64_t *validity, idx_t row);

/*!
In a validity mask, sets a specific row to valid.

Equivalent to `duckdb_validity_set_row_validity` with valid set to true.

* validity: The validity mask
* row: The row index
*/
DUCKDB_API void duckdb_validity_set_row_valid(uint64_t *validity, idx_t row);

//===--------------------------------------------------------------------===//
// Table Functions
//===--------------------------------------------------------------------===//
typedef void *duckdb_table_function;
typedef void *duckdb_bind_info;
typedef void *duckdb_init_info;
typedef void *duckdb_function_info;

typedef void (*duckdb_table_function_bind_t)(duckdb_bind_info info);
typedef void (*duckdb_table_function_init_t)(duckdb_init_info info);
typedef void (*duckdb_table_function_t)(duckdb_function_info info, duckdb_data_chunk output);
typedef void (*duckdb_delete_callback_t)(void *data);

/*!
Creates a new empty table function.

The return value should be destroyed with `duckdb_destroy_table_function`.

* returns: The table function object.
*/
DUCKDB_API duckdb_table_function duckdb_create_table_function();

/*!
Destroys the given table function object.

* table_function: The table function to destroy
*/
DUCKDB_API void duckdb_destroy_table_function(duckdb_table_function *table_function);

/*!
Sets the name of the given table function.

* table_function: The table function
* name: The name of the table function
*/
DUCKDB_API void duckdb_table_function_set_name(duckdb_table_function table_function, const char *name);

/*!
Adds a parameter to the table function.

* table_function: The table function
* type: The type of the parameter to add.
*/
DUCKDB_API void duckdb_table_function_add_parameter(duckdb_table_function table_function, duckdb_logical_type type);

/*!
Adds a named parameter to the table function.

* table_function: The table function
* name: The name of the parameter
* type: The type of the parameter to add.
*/
DUCKDB_API void duckdb_table_function_add_named_parameter(duckdb_table_function table_function, const char *name,
                                                          duckdb_logical_type type);

/*!
Assigns extra information to the table function that can be fetched during binding, etc.

* table_function: The table function
* extra_info: The extra information
* destroy: The callback that will be called to destroy the bind data (if any)
*/
DUCKDB_API void duckdb_table_function_set_extra_info(duckdb_table_function table_function, void *extra_info,
                                                     duckdb_delete_callback_t destroy);

/*!
Sets the bind function of the table function

* table_function: The table function
* bind: The bind function
*/
DUCKDB_API void duckdb_table_function_set_bind(duckdb_table_function table_function, duckdb_table_function_bind_t bind);

/*!
Sets the init function of the table function

* table_function: The table function
* init: The init function
*/
DUCKDB_API void duckdb_table_function_set_init(duckdb_table_function table_function, duckdb_table_function_init_t init);

/*!
Sets the thread-local init function of the table function

* table_function: The table function
* init: The init function
*/
DUCKDB_API void duckdb_table_function_set_local_init(duckdb_table_function table_function,
                                                     duckdb_table_function_init_t init);

/*!
Sets the main function of the table function

* table_function: The table function
* function: The function
*/
DUCKDB_API void duckdb_table_function_set_function(duckdb_table_function table_function,
                                                   duckdb_table_function_t function);

/*!
Sets whether or not the given table function supports projection pushdown.

If this is set to true, the system will provide a list of all required columns in the `init` stage through
the `duckdb_init_get_column_count` and `duckdb_init_get_column_index` functions.
If this is set to false (the default), the system will expect all columns to be projected.

* table_function: The table function
* pushdown: True if the table function supports projection pushdown, false otherwise.
*/
DUCKDB_API void duckdb_table_function_supports_projection_pushdown(duckdb_table_function table_function, bool pushdown);

/*!
Register the table function object within the given connection.

The function requires at least a name, a bind function, an init function and a main function.

If the function is incomplete or a function with this name already exists DuckDBError is returned.

* con: The connection to register it in.
* function: The function pointer
* returns: Whether or not the registration was successful.
*/
DUCKDB_API duckdb_state duckdb_register_table_function(duckdb_connection con, duckdb_table_function function);

//===--------------------------------------------------------------------===//
// Table Function Bind
//===--------------------------------------------------------------------===//
/*!
Retrieves the extra info of the function as set in `duckdb_table_function_set_extra_info`

* info: The info object
* returns: The extra info
*/
DUCKDB_API void *duckdb_bind_get_extra_info(duckdb_bind_info info);

/*!
Adds a result column to the output of the table function.

* info: The info object
* name: The name of the column
* type: The logical type of the column
*/
DUCKDB_API void duckdb_bind_add_result_column(duckdb_bind_info info, const char *name, duckdb_logical_type type);

/*!
Retrieves the number of regular (non-named) parameters to the function.

* info: The info object
* returns: The number of parameters
*/
DUCKDB_API idx_t duckdb_bind_get_parameter_count(duckdb_bind_info info);

/*!
Retrieves the parameter at the given index.

The result must be destroyed with `duckdb_destroy_value`.

* info: The info object
* index: The index of the parameter to get
* returns: The value of the parameter. Must be destroyed with `duckdb_destroy_value`.
*/
DUCKDB_API duckdb_value duckdb_bind_get_parameter(duckdb_bind_info info, idx_t index);

/*!
Retrieves a named parameter with the given name.

The result must be destroyed with `duckdb_destroy_value`.

* info: The info object
* name: The name of the parameter
* returns: The value of the parameter. Must be destroyed with `duckdb_destroy_value`.
*/
DUCKDB_API duckdb_value duckdb_bind_get_named_parameter(duckdb_bind_info info, const char *name);

/*!
Sets the user-provided bind data in the bind object. This object can be retrieved again during execution.

* info: The info object
* extra_data: The bind data object.
* destroy: The callback that will be called to destroy the bind data (if any)
*/
DUCKDB_API void duckdb_bind_set_bind_data(duckdb_bind_info info, void *bind_data, duckdb_delete_callback_t destroy);

/*!
Sets the cardinality estimate for the table function, used for optimization.

* info: The bind data object.
* is_exact: Whether or not the cardinality estimate is exact, or an approximation
*/
DUCKDB_API void duckdb_bind_set_cardinality(duckdb_bind_info info, idx_t cardinality, bool is_exact);

/*!
Report that an error has occurred while calling bind.

* info: The info object
* error: The error message
*/
DUCKDB_API void duckdb_bind_set_error(duckdb_bind_info info, const char *error);

//===--------------------------------------------------------------------===//
// Table Function Init
//===--------------------------------------------------------------------===//

/*!
Retrieves the extra info of the function as set in `duckdb_table_function_set_extra_info`

* info: The info object
* returns: The extra info
*/
DUCKDB_API void *duckdb_init_get_extra_info(duckdb_init_info info);

/*!
Gets the bind data set by `duckdb_bind_set_bind_data` during the bind.

Note that the bind data should be considered as read-only.
For tracking state, use the init data instead.

* info: The info object
* returns: The bind data object
*/
DUCKDB_API void *duckdb_init_get_bind_data(duckdb_init_info info);

/*!
Sets the user-provided init data in the init object. This object can be retrieved again during execution.

* info: The info object
* extra_data: The init data object.
* destroy: The callback that will be called to destroy the init data (if any)
*/
DUCKDB_API void duckdb_init_set_init_data(duckdb_init_info info, void *init_data, duckdb_delete_callback_t destroy);

/*!
Returns the number of projected columns.

This function must be used if projection pushdown is enabled to figure out which columns to emit.

* info: The info object
* returns: The number of projected columns.
*/
DUCKDB_API idx_t duckdb_init_get_column_count(duckdb_init_info info);

/*!
Returns the column index of the projected column at the specified position.

This function must be used if projection pushdown is enabled to figure out which columns to emit.

* info: The info object
* column_index: The index at which to get the projected column index, from 0..duckdb_init_get_column_count(info)
* returns: The column index of the projected column.
*/
DUCKDB_API idx_t duckdb_init_get_column_index(duckdb_init_info info, idx_t column_index);

/*!
Sets how many threads can process this table function in parallel (default: 1)

* info: The info object
* max_threads: The maximum amount of threads that can process this table function
*/
DUCKDB_API void duckdb_init_set_max_threads(duckdb_init_info info, idx_t max_threads);

/*!
Report that an error has occurred while calling init.

* info: The info object
* error: The error message
*/
DUCKDB_API void duckdb_init_set_error(duckdb_init_info info, const char *error);

//===--------------------------------------------------------------------===//
// Table Function
//===--------------------------------------------------------------------===//

/*!
Retrieves the extra info of the function as set in `duckdb_table_function_set_extra_info`

* info: The info object
* returns: The extra info
*/
DUCKDB_API void *duckdb_function_get_extra_info(duckdb_function_info info);
/*!
Gets the bind data set by `duckdb_bind_set_bind_data` during the bind.

Note that the bind data should be considered as read-only.
For tracking state, use the init data instead.

* info: The info object
* returns: The bind data object
*/
DUCKDB_API void *duckdb_function_get_bind_data(duckdb_function_info info);

/*!
Gets the init data set by `duckdb_init_set_init_data` during the init.

* info: The info object
* returns: The init data object
*/
DUCKDB_API void *duckdb_function_get_init_data(duckdb_function_info info);

/*!
Gets the thread-local init data set by `duckdb_init_set_init_data` during the local_init.

* info: The info object
* returns: The init data object
*/
DUCKDB_API void *duckdb_function_get_local_init_data(duckdb_function_info info);

/*!
Report that an error has occurred while executing the function.

* info: The info object
* error: The error message
*/
DUCKDB_API void duckdb_function_set_error(duckdb_function_info info, const char *error);

//===--------------------------------------------------------------------===//
// Replacement Scans
//===--------------------------------------------------------------------===//
typedef void *duckdb_replacement_scan_info;

typedef void (*duckdb_replacement_callback_t)(duckdb_replacement_scan_info info, const char *table_name, void *data);

/*!
Add a replacement scan definition to the specified database

* db: The database object to add the replacement scan to
* replacement: The replacement scan callback
* extra_data: Extra data that is passed back into the specified callback
* delete_callback: The delete callback to call on the extra data, if any
*/
DUCKDB_API void duckdb_add_replacement_scan(duckdb_database db, duckdb_replacement_callback_t replacement,
                                            void *extra_data, duckdb_delete_callback_t delete_callback);

/*!
Sets the replacement function name to use. If this function is called in the replacement callback,
 the replacement scan is performed. If it is not called, the replacement callback is not performed.

* info: The info object
* function_name: The function name to substitute.
*/
DUCKDB_API void duckdb_replacement_scan_set_function_name(duckdb_replacement_scan_info info, const char *function_name);

/*!
Adds a parameter to the replacement scan function.

* info: The info object
* parameter: The parameter to add.
*/
DUCKDB_API void duckdb_replacement_scan_add_parameter(duckdb_replacement_scan_info info, duckdb_value parameter);

/*!
Report that an error has occurred while executing the replacement scan.

* info: The info object
* error: The error message
*/
DUCKDB_API void duckdb_replacement_scan_set_error(duckdb_replacement_scan_info info, const char *error);

//===--------------------------------------------------------------------===//
// Appender
//===--------------------------------------------------------------------===//

// Appenders are the most efficient way of loading data into DuckDB from within the C interface, and are recommended for
// fast data loading. The appender is much faster than using prepared statements or individual `INSERT INTO` statements.

// Appends are made in row-wise format. For every column, a `duckdb_append_[type]` call should be made, after which
// the row should be finished by calling `duckdb_appender_end_row`. After all rows have been appended,
// `duckdb_appender_destroy` should be used to finalize the appender and clean up the resulting memory.

// Note that `duckdb_appender_destroy` should always be called on the resulting appender, even if the function returns
// `DuckDBError`.

/*!
Creates an appender object.

* connection: The connection context to create the appender in.
* schema: The schema of the table to append to, or `nullptr` for the default schema.
* table: The table name to append to.
* out_appender: The resulting appender object.
* returns: `DuckDBSuccess` on success or `DuckDBError` on failure.
*/
DUCKDB_API duckdb_state duckdb_appender_create(duckdb_connection connection, const char *schema, const char *table,
                                               duckdb_appender *out_appender);

/*!
Returns the error message associated with the given appender.
If the appender has no error message, this returns `nullptr` instead.

The error message should not be freed. It will be de-allocated when `duckdb_appender_destroy` is called.

* appender: The appender to get the error from.
* returns: The error message, or `nullptr` if there is none.
*/
DUCKDB_API const char *duckdb_appender_error(duckdb_appender appender);

/*!
Flush the appender to the table, forcing the cache of the appender to be cleared and the data to be appended to the
base table.

This should generally not be used unless you know what you are doing. Instead, call `duckdb_appender_destroy` when you
are done with the appender.

* appender: The appender to flush.
* returns: `DuckDBSuccess` on success or `DuckDBError` on failure.
*/
DUCKDB_API duckdb_state duckdb_appender_flush(duckdb_appender appender);

/*!
Close the appender, flushing all intermediate state in the appender to the table and closing it for further appends.

This is generally not necessary. Call `duckdb_appender_destroy` instead.

* appender: The appender to flush and close.
* returns: `DuckDBSuccess` on success or `DuckDBError` on failure.
*/
DUCKDB_API duckdb_state duckdb_appender_close(duckdb_appender appender);

/*!
Close the appender and destroy it. Flushing all intermediate state in the appender to the table, and de-allocating
all memory associated with the appender.

* appender: The appender to flush, close and destroy.
* returns: `DuckDBSuccess` on success or `DuckDBError` on failure.
*/
DUCKDB_API duckdb_state duckdb_appender_destroy(duckdb_appender *appender);

/*!
A nop function, provided for backwards compatibility reasons. Does nothing. Only `duckdb_appender_end_row` is required.
*/
DUCKDB_API duckdb_state duckdb_appender_begin_row(duckdb_appender appender);

/*!
Finish the current row of appends. After end_row is called, the next row can be appended.

* appender: The appender.
* returns: `DuckDBSuccess` on success or `DuckDBError` on failure.
*/
DUCKDB_API duckdb_state duckdb_appender_end_row(duckdb_appender appender);

/*!
Append a bool value to the appender.
*/
DUCKDB_API duckdb_state duckdb_append_bool(duckdb_appender appender, bool value);

/*!
Append an int8_t value to the appender.
*/
DUCKDB_API duckdb_state duckdb_append_int8(duckdb_appender appender, int8_t value);
/*!
Append an int16_t value to the appender.
*/
DUCKDB_API duckdb_state duckdb_append_int16(duckdb_appender appender, int16_t value);
/*!
Append an int32_t value to the appender.
*/
DUCKDB_API duckdb_state duckdb_append_int32(duckdb_appender appender, int32_t value);
/*!
Append an int64_t value to the appender.
*/
DUCKDB_API duckdb_state duckdb_append_int64(duckdb_appender appender, int64_t value);
/*!
Append a duckdb_hugeint value to the appender.
*/
DUCKDB_API duckdb_state duckdb_append_hugeint(duckdb_appender appender, duckdb_hugeint value);

/*!
Append a uint8_t value to the appender.
*/
DUCKDB_API duckdb_state duckdb_append_uint8(duckdb_appender appender, uint8_t value);
/*!
Append a uint16_t value to the appender.
*/
DUCKDB_API duckdb_state duckdb_append_uint16(duckdb_appender appender, uint16_t value);
/*!
Append a uint32_t value to the appender.
*/
DUCKDB_API duckdb_state duckdb_append_uint32(duckdb_appender appender, uint32_t value);
/*!
Append a uint64_t value to the appender.
*/
DUCKDB_API duckdb_state duckdb_append_uint64(duckdb_appender appender, uint64_t value);

/*!
Append a float value to the appender.
*/
DUCKDB_API duckdb_state duckdb_append_float(duckdb_appender appender, float value);
/*!
Append a double value to the appender.
*/
DUCKDB_API duckdb_state duckdb_append_double(duckdb_appender appender, double value);

/*!
Append a duckdb_date value to the appender.
*/
DUCKDB_API duckdb_state duckdb_append_date(duckdb_appender appender, duckdb_date value);
/*!
Append a duckdb_time value to the appender.
*/
DUCKDB_API duckdb_state duckdb_append_time(duckdb_appender appender, duckdb_time value);
/*!
Append a duckdb_timestamp value to the appender.
*/
DUCKDB_API duckdb_state duckdb_append_timestamp(duckdb_appender appender, duckdb_timestamp value);
/*!
Append a duckdb_interval value to the appender.
*/
DUCKDB_API duckdb_state duckdb_append_interval(duckdb_appender appender, duckdb_interval value);

/*!
Append a varchar value to the appender.
*/
DUCKDB_API duckdb_state duckdb_append_varchar(duckdb_appender appender, const char *val);
/*!
Append a varchar value to the appender.
*/
DUCKDB_API duckdb_state duckdb_append_varchar_length(duckdb_appender appender, const char *val, idx_t length);
/*!
Append a blob value to the appender.
*/
DUCKDB_API duckdb_state duckdb_append_blob(duckdb_appender appender, const void *data, idx_t length);
/*!
Append a NULL value to the appender (of any type).
*/
DUCKDB_API duckdb_state duckdb_append_null(duckdb_appender appender);

/*!
Appends a pre-filled data chunk to the specified appender.

The types of the data chunk must exactly match the types of the table, no casting is performed.
If the types do not match or the appender is in an invalid state, DuckDBError is returned.
If the append is successful, DuckDBSuccess is returned.

* appender: The appender to append to.
* chunk: The data chunk to append.
* returns: The return state.
*/
DUCKDB_API duckdb_state duckdb_append_data_chunk(duckdb_appender appender, duckdb_data_chunk chunk);

//===--------------------------------------------------------------------===//
// Arrow Interface
//===--------------------------------------------------------------------===//
/*!
Executes a SQL query within a connection and stores the full (materialized) result in an arrow structure.
If the query fails to execute, DuckDBError is returned and the error message can be retrieved by calling
`duckdb_query_arrow_error`.

Note that after running `duckdb_query_arrow`, `duckdb_destroy_arrow` must be called on the result object even if the
query fails, otherwise the error stored within the result will not be freed correctly.

* connection: The connection to perform the query in.
* query: The SQL query to run.
* out_result: The query result.
* returns: `DuckDBSuccess` on success or `DuckDBError` on failure.
*/
DUCKDB_API duckdb_state duckdb_query_arrow(duckdb_connection connection, const char *query, duckdb_arrow *out_result);

/*!
Fetch the internal arrow schema from the arrow result.

* result: The result to fetch the schema from.
* out_schema: The output schema.
* returns: `DuckDBSuccess` on success or `DuckDBError` on failure.
*/
DUCKDB_API duckdb_state duckdb_query_arrow_schema(duckdb_arrow result, duckdb_arrow_schema *out_schema);

/*!
Fetch an internal arrow array from the arrow result.

This function can be called multiple time to get next chunks, which will free the previous out_array.
So consume the out_array before calling this function again.

* result: The result to fetch the array from.
* out_array: The output array.
* returns: `DuckDBSuccess` on success or `DuckDBError` on failure.
*/
DUCKDB_API duckdb_state duckdb_query_arrow_array(duckdb_arrow result, duckdb_arrow_array *out_array);

/*!
Returns the number of columns present in a the arrow result object.

* result: The result object.
* returns: The number of columns present in the result object.
*/
DUCKDB_API idx_t duckdb_arrow_column_count(duckdb_arrow result);

/*!
Returns the number of rows present in a the arrow result object.

* result: The result object.
* returns: The number of rows present in the result object.
*/
DUCKDB_API idx_t duckdb_arrow_row_count(duckdb_arrow result);

/*!
Returns the number of rows changed by the query stored in the arrow result. This is relevant only for
INSERT/UPDATE/DELETE queries. For other queries the rows_changed will be 0.

* result: The result object.
* returns: The number of rows changed.
*/
DUCKDB_API idx_t duckdb_arrow_rows_changed(duckdb_arrow result);

/*!
Returns the error message contained within the result. The error is only set if `duckdb_query_arrow` returns
`DuckDBError`.

The error message should not be freed. It will be de-allocated when `duckdb_destroy_arrow` is called.

* result: The result object to fetch the nullmask from.
* returns: The error of the result.
*/
DUCKDB_API const char *duckdb_query_arrow_error(duckdb_arrow result);

/*!
Closes the result and de-allocates all memory allocated for the arrow result.

* result: The result to destroy.
*/
DUCKDB_API void duckdb_destroy_arrow(duckdb_arrow *result);

//===--------------------------------------------------------------------===//
// Threading Information
//===--------------------------------------------------------------------===//
typedef void *duckdb_task_state;

/*!
Execute DuckDB tasks on this thread.

Will return after `max_tasks` have been executed, or if there are no more tasks present.

* database: The database object to execute tasks for
* max_tasks: The maximum amount of tasks to execute
*/
DUCKDB_API void duckdb_execute_tasks(duckdb_database database, idx_t max_tasks);

/*!
Creates a task state that can be used with duckdb_execute_tasks_state to execute tasks until
 duckdb_finish_execution is called on the state.

duckdb_destroy_state should be called on the result in order to free memory.

* database: The database object to create the task state for
* returns: The task state that can be used with duckdb_execute_tasks_state.
*/
DUCKDB_API duckdb_task_state duckdb_create_task_state(duckdb_database database);

/*!
Execute DuckDB tasks on this thread.

The thread will keep on executing tasks forever, until duckdb_finish_execution is called on the state.
Multiple threads can share the same duckdb_task_state.

* state: The task state of the executor
*/
DUCKDB_API void duckdb_execute_tasks_state(duckdb_task_state state);

/*!
Execute DuckDB tasks on this thread.

The thread will keep on executing tasks until either duckdb_finish_execution is called on the state,
max_tasks tasks have been executed or there are no more tasks to be executed.

Multiple threads can share the same duckdb_task_state.

* state: The task state of the executor
* max_tasks: The maximum amount of tasks to execute
* returns: The amount of tasks that have actually been executed
*/
DUCKDB_API idx_t duckdb_execute_n_tasks_state(duckdb_task_state state, idx_t max_tasks);

/*!
Finish execution on a specific task.

* state: The task state to finish execution
*/
DUCKDB_API void duckdb_finish_execution(duckdb_task_state state);

/*!
Check if the provided duckdb_task_state has finished execution

* state: The task state to inspect
* returns: Whether or not duckdb_finish_execution has been called on the task state
*/
DUCKDB_API bool duckdb_task_state_is_finished(duckdb_task_state state);

/*!
Destroys the task state returned from duckdb_create_task_state.

Note that this should not be called while there is an active duckdb_execute_tasks_state running
on the task state.

* state: The task state to clean up
*/
DUCKDB_API void duckdb_destroy_task_state(duckdb_task_state state);

/*!
Returns true if execution of the current query is finished.

* con: The connection on which to check
*/
DUCKDB_API bool duckdb_execution_is_finished(duckdb_connection con);

//===--------------------------------------------------------------------===//
// Streaming Result Interface
//===--------------------------------------------------------------------===//

/*!
Fetches a data chunk from the (streaming) duckdb_result. This function should be called repeatedly until the result is
exhausted.

The result must be destroyed with `duckdb_destroy_data_chunk`.

This function can only be used on duckdb_results created with 'duckdb_pending_prepared_streaming'

If this function is used, none of the other result functions can be used and vice versa (i.e. this function cannot be
mixed with the legacy result functions or the materialized result functions).

It is not known beforehand how many chunks will be returned by this result.

* result: The result object to fetch the data chunk from.
* returns: The resulting data chunk. Returns `NULL` if the result has an error.
*/
DUCKDB_API duckdb_data_chunk duckdb_stream_fetch_chunk(duckdb_result result);

#ifdef __cplusplus
}
#endif



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/profiler.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {

//! The profiler can be used to measure elapsed time
template <typename T>
class BaseProfiler {
public:
	//! Starts the timer
	void Start() {
		finished = false;
		start = Tick();
	}
	//! Finishes timing
	void End() {
		end = Tick();
		finished = true;
	}

	//! Returns the elapsed time in seconds. If End() has been called, returns
	//! the total elapsed time. Otherwise returns how far along the timer is
	//! right now.
	double Elapsed() const {
		auto _end = finished ? end : Tick();
		return std::chrono::duration_cast<std::chrono::duration<double>>(_end - start).count();
	}

private:
	time_point<T> Tick() const {
		return T::now();
	}
	time_point<T> start;
	time_point<T> end;
	bool finished = false;
};

using Profiler = BaseProfiler<system_clock>;

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/progress_bar/progress_bar_display.hpp
//
//
//===----------------------------------------------------------------------===//



namespace duckdb {

class ProgressBarDisplay {
public:
	ProgressBarDisplay() {
	}
	virtual ~ProgressBarDisplay() {
	}

public:
	virtual void Update(double percentage) = 0;
	virtual void Finish() = 0;
};

} // namespace duckdb


namespace duckdb {

struct ClientConfig;
typedef unique_ptr<ProgressBarDisplay> (*progress_bar_display_create_func_t)();

class ProgressBar {
public:
	static unique_ptr<ProgressBarDisplay> DefaultProgressBarDisplay();
	static void SystemOverrideCheck(ClientConfig &config);

	explicit ProgressBar(
	    Executor &executor, idx_t show_progress_after,
	    progress_bar_display_create_func_t create_display_func = ProgressBar::DefaultProgressBarDisplay);

	//! Starts the thread
	void Start();
	//! Updates the progress bar and prints it to the screen
	void Update(bool final);
	//! Gets current percentage
	double GetCurrentPercentage();

	void PrintProgressInternal(int percentage);
	void PrintProgress(int percentage);
	void FinishProgressBarPrint();
	bool ShouldPrint(bool final) const;
	bool PrintEnabled() const;

private:
	//! The executor
	Executor &executor;
	//! The profiler used to measure the time since the progress bar was started
	Profiler profiler;
	//! The time in ms after which to start displaying the progress bar
	idx_t show_progress_after;
	//! The current progress percentage
	double current_percentage;
	//! The display used to print the progress
	unique_ptr<ProgressBarDisplay> display;
	//! Whether or not profiling is supported for the current query
	bool supported = true;
	//! Whether the bar has already finished
	bool finished = false;
};
} // namespace duckdb


namespace duckdb {
class ClientContext;
class PhysicalResultCollector;
class PreparedStatementData;

typedef std::function<unique_ptr<PhysicalResultCollector>(ClientContext &context, PreparedStatementData &data)>
    get_result_collector_t;

struct ClientConfig {
	//! The home directory used by the system (if any)
	string home_directory;
	//! If the query profiler is enabled or not.
	bool enable_profiler = false;
	//! If detailed query profiling is enabled
	bool enable_detailed_profiling = false;
	//! The format to print query profiling information in (default: query_tree), if enabled.
	ProfilerPrintFormat profiler_print_format = ProfilerPrintFormat::QUERY_TREE;
	//! The file to save query profiling information to, instead of printing it to the console
	//! (empty = print to console)
	string profiler_save_location;

	//! Allows suppressing profiler output, even if enabled. We turn on the profiler on all test runs but don't want
	//! to output anything
	bool emit_profiler_output = true;

	//! system-wide progress bar disable.
	const char *system_progress_bar_disable_reason = nullptr;
	//! If the progress bar is enabled or not.
	bool enable_progress_bar = false;
	//! If the print of the progress bar is enabled
	bool print_progress_bar = true;
	//! The wait time before showing the progress bar
	int wait_time = 2000;

	//! Preserve identifier case while parsing.
	//! If false, all unquoted identifiers are lower-cased (e.g. "MyTable" -> "mytable").
	bool preserve_identifier_case = true;
	//! The maximum expression depth limit in the parser
	idx_t max_expression_depth = 1000;

	//! Whether or not aggressive query verification is enabled
	bool query_verification_enabled = false;
	//! Whether or not verification of external operators is enabled, used for testing
	bool verify_external = false;
	//! Whether or not we should verify the serializer
	bool verify_serializer = false;
	//! Enable the running of optimizers
	bool enable_optimizer = true;
	//! Enable caching operators
	bool enable_caching_operators = true;
	//! Force parallelism of small tables, used for testing
	bool verify_parallelism = false;
	//! Force index join independent of table cardinality, used for testing
	bool force_index_join = false;
	//! Force out-of-core computation for operators that support it, used for testing
	bool force_external = false;
	//! Force disable cross product generation when hyper graph isn't connected, used for testing
	bool force_no_cross_product = false;
	//! Force use of IEJoin to implement AsOfJoin, used for testing
	bool force_asof_iejoin = false;
	//! If this context should also try to use the available replacement scans
	//! True by default
	bool use_replacement_scans = true;
	//! Maximum bits allowed for using a perfect hash table (i.e. the perfect HT can hold up to 2^perfect_ht_threshold
	//! elements)
	idx_t perfect_ht_threshold = 12;
	//! The maximum number of rows to accumulate before sorting ordered aggregates.
	idx_t ordered_aggregate_threshold = (idx_t(1) << 18);

	//! Callback to create a progress bar display
	progress_bar_display_create_func_t display_create_func = nullptr;

	//! Override for the default extension repository
	string custom_extension_repo = "";

	//! The explain output type used when none is specified (default: PHYSICAL_ONLY)
	ExplainOutputType explain_output_type = ExplainOutputType::PHYSICAL_ONLY;

	//! The maximum amount of pivot columns
	idx_t pivot_limit = 100000;

	//! Whether or not the "/" division operator defaults to integer division or floating point division
	bool integer_division = false;

	//! Generic options
	case_insensitive_map_t<Value> set_variables;

	//! Function that is used to create the result collector for a materialized result
	//! Defaults to PhysicalMaterializedCollector
	get_result_collector_t result_collector = nullptr;

public:
	static ClientConfig &GetConfig(ClientContext &context);
	static const ClientConfig &GetConfig(const ClientContext &context);

	string ExtractTimezone() const;

	bool AnyVerification() {
		return query_verification_enabled || verify_external || verify_serializer;
	}
};

} // namespace duckdb




namespace duckdb {
class Appender;
class Catalog;
class CatalogSearchPath;
class ColumnDataCollection;
class DatabaseInstance;
class FileOpener;
class LogicalOperator;
class PreparedStatementData;
class Relation;
class BufferedFileWriter;
class QueryProfiler;
class ClientContextLock;
struct CreateScalarFunctionInfo;
class ScalarFunctionCatalogEntry;
struct ActiveQueryContext;
struct ParserOptions;
struct ClientData;

struct PendingQueryParameters {
	//! Prepared statement parameters (if any)
	vector<Value> *parameters = nullptr;
	//! Whether or not a stream result should be allowed
	bool allow_stream_result = false;
};

//! ClientContextState is virtual base class for ClientContext-local (or Query-Local, using QueryEnd callback) state
//! e.g. caches that need to live as long as a ClientContext or Query.
class ClientContextState {
public:
	virtual ~ClientContextState() {};
	virtual void QueryEnd() = 0;
};

//! The ClientContext holds information relevant to the current client session
//! during execution
class ClientContext : public std::enable_shared_from_this<ClientContext> {
	friend class PendingQueryResult;
	friend class StreamQueryResult;
	friend class DuckTransactionManager;

public:
	DUCKDB_API explicit ClientContext(shared_ptr<DatabaseInstance> db);
	DUCKDB_API ~ClientContext();

	//! The database that this client is connected to
	shared_ptr<DatabaseInstance> db;
	//! Whether or not the query is interrupted
	atomic<bool> interrupted;
	//! External Objects (e.g., Python objects) that views depend of
	unordered_map<string, vector<shared_ptr<ExternalDependency>>> external_dependencies;
	//! Set of optional states (e.g. Caches) that can be held by the ClientContext
	unordered_map<string, shared_ptr<ClientContextState>> registered_state;
	//! The client configuration
	ClientConfig config;
	//! The set of client-specific data
	unique_ptr<ClientData> client_data;
	//! Data for the currently running transaction
	TransactionContext transaction;

public:
	MetaTransaction &ActiveTransaction() {
		return transaction.ActiveTransaction();
	}

	//! Interrupt execution of a query
	DUCKDB_API void Interrupt();
	//! Enable query profiling
	DUCKDB_API void EnableProfiling();
	//! Disable query profiling
	DUCKDB_API void DisableProfiling();

	//! Issue a query, returning a QueryResult. The QueryResult can be either a StreamQueryResult or a
	//! MaterializedQueryResult. The StreamQueryResult will only be returned in the case of a successful SELECT
	//! statement.
	DUCKDB_API unique_ptr<QueryResult> Query(const string &query, bool allow_stream_result);
	DUCKDB_API unique_ptr<QueryResult> Query(unique_ptr<SQLStatement> statement, bool allow_stream_result);

	//! Issues a query to the database and returns a Pending Query Result. Note that "query" may only contain
	//! a single statement.
	DUCKDB_API unique_ptr<PendingQueryResult> PendingQuery(const string &query, bool allow_stream_result);
	//! Issues a query to the database and returns a Pending Query Result
	DUCKDB_API unique_ptr<PendingQueryResult> PendingQuery(unique_ptr<SQLStatement> statement,
	                                                       bool allow_stream_result);

	//! Destroy the client context
	DUCKDB_API void Destroy();

	//! Get the table info of a specific table, or nullptr if it cannot be found
	DUCKDB_API unique_ptr<TableDescription> TableInfo(const string &schema_name, const string &table_name);
	//! Appends a DataChunk to the specified table. Returns whether or not the append was successful.
	DUCKDB_API void Append(TableDescription &description, ColumnDataCollection &collection);
	//! Try to bind a relation in the current client context; either throws an exception or fills the result_columns
	//! list with the set of returned columns
	DUCKDB_API void TryBindRelation(Relation &relation, vector<ColumnDefinition> &result_columns);

	//! Execute a relation
	DUCKDB_API unique_ptr<PendingQueryResult> PendingQuery(const shared_ptr<Relation> &relation,
	                                                       bool allow_stream_result);
	DUCKDB_API unique_ptr<QueryResult> Execute(const shared_ptr<Relation> &relation);

	//! Prepare a query
	DUCKDB_API unique_ptr<PreparedStatement> Prepare(const string &query);
	//! Directly prepare a SQL statement
	DUCKDB_API unique_ptr<PreparedStatement> Prepare(unique_ptr<SQLStatement> statement);

	//! Create a pending query result from a prepared statement with the given name and set of parameters
	//! It is possible that the prepared statement will be re-bound. This will generally happen if the catalog is
	//! modified in between the prepared statement being bound and the prepared statement being run.
	DUCKDB_API unique_ptr<PendingQueryResult>
	PendingQuery(const string &query, shared_ptr<PreparedStatementData> &prepared, PendingQueryParameters parameters);

	//! Execute a prepared statement with the given name and set of parameters
	//! It is possible that the prepared statement will be re-bound. This will generally happen if the catalog is
	//! modified in between the prepared statement being bound and the prepared statement being run.
	DUCKDB_API unique_ptr<QueryResult> Execute(const string &query, shared_ptr<PreparedStatementData> &prepared,
	                                           vector<Value> &values, bool allow_stream_result = true);
	DUCKDB_API unique_ptr<QueryResult> Execute(const string &query, shared_ptr<PreparedStatementData> &prepared,
	                                           PendingQueryParameters parameters);

	//! Gets current percentage of the query's progress, returns 0 in case the progress bar is disabled.
	DUCKDB_API double GetProgress();

	//! Register function in the temporary schema
	DUCKDB_API void RegisterFunction(CreateFunctionInfo &info);

	//! Parse statements from a query
	DUCKDB_API vector<unique_ptr<SQLStatement>> ParseStatements(const string &query);

	//! Extract the logical plan of a query
	DUCKDB_API unique_ptr<LogicalOperator> ExtractPlan(const string &query);
	DUCKDB_API void HandlePragmaStatements(vector<unique_ptr<SQLStatement>> &statements);

	//! Runs a function with a valid transaction context, potentially starting a transaction if the context is in auto
	//! commit mode.
	DUCKDB_API void RunFunctionInTransaction(const std::function<void(void)> &fun,
	                                         bool requires_valid_transaction = true);
	//! Same as RunFunctionInTransaction, but does not obtain a lock on the client context or check for validation
	DUCKDB_API void RunFunctionInTransactionInternal(ClientContextLock &lock, const std::function<void(void)> &fun,
	                                                 bool requires_valid_transaction = true);

	//! Equivalent to CURRENT_SETTING(key) SQL function.
	DUCKDB_API bool TryGetCurrentSetting(const std::string &key, Value &result);

	//! Returns the parser options for this client context
	DUCKDB_API ParserOptions GetParserOptions() const;

	DUCKDB_API unique_ptr<DataChunk> Fetch(ClientContextLock &lock, StreamQueryResult &result);

	//! Whether or not the given result object (streaming query result or pending query result) is active
	DUCKDB_API bool IsActiveResult(ClientContextLock &lock, BaseQueryResult *result);

	//! Returns the current executor
	Executor &GetExecutor();

	//! Returns the current query string (if any)
	const string &GetCurrentQuery();

	//! Fetch a list of table names that are required for a given query
	DUCKDB_API unordered_set<string> GetTableNames(const string &query);

	DUCKDB_API ClientProperties GetClientProperties() const;

	//! Returns true if execution of the current query is finished
	DUCKDB_API bool ExecutionIsFinished();

private:
	//! Parse statements and resolve pragmas from a query
	bool ParseStatements(ClientContextLock &lock, const string &query, vector<unique_ptr<SQLStatement>> &result,
	                     PreservedError &error);
	//! Issues a query to the database and returns a Pending Query Result
	unique_ptr<PendingQueryResult> PendingQueryInternal(ClientContextLock &lock, unique_ptr<SQLStatement> statement,
	                                                    PendingQueryParameters parameters, bool verify = true);
	unique_ptr<QueryResult> ExecutePendingQueryInternal(ClientContextLock &lock, PendingQueryResult &query);

	//! Parse statements from a query
	vector<unique_ptr<SQLStatement>> ParseStatementsInternal(ClientContextLock &lock, const string &query);
	//! Perform aggressive query verification of a SELECT statement. Only called when query_verification_enabled is
	//! true.
	PreservedError VerifyQuery(ClientContextLock &lock, const string &query, unique_ptr<SQLStatement> statement);

	void InitialCleanup(ClientContextLock &lock);
	//! Internal clean up, does not lock. Caller must hold the context_lock.
	void CleanupInternal(ClientContextLock &lock, BaseQueryResult *result = nullptr,
	                     bool invalidate_transaction = false);
	unique_ptr<PendingQueryResult> PendingStatementOrPreparedStatement(ClientContextLock &lock, const string &query,
	                                                                   unique_ptr<SQLStatement> statement,
	                                                                   shared_ptr<PreparedStatementData> &prepared,
	                                                                   PendingQueryParameters parameters);
	unique_ptr<PendingQueryResult> PendingPreparedStatement(ClientContextLock &lock,
	                                                        shared_ptr<PreparedStatementData> statement_p,
	                                                        PendingQueryParameters parameters);

	//! Internally prepare a SQL statement. Caller must hold the context_lock.
	shared_ptr<PreparedStatementData> CreatePreparedStatement(ClientContextLock &lock, const string &query,
	                                                          unique_ptr<SQLStatement> statement,
	                                                          vector<Value> *values = nullptr);
	unique_ptr<PendingQueryResult> PendingStatementInternal(ClientContextLock &lock, const string &query,
	                                                        unique_ptr<SQLStatement> statement,
	                                                        PendingQueryParameters parameters);
	unique_ptr<QueryResult> RunStatementInternal(ClientContextLock &lock, const string &query,
	                                             unique_ptr<SQLStatement> statement, bool allow_stream_result,
	                                             bool verify = true);
	unique_ptr<PreparedStatement> PrepareInternal(ClientContextLock &lock, unique_ptr<SQLStatement> statement);
	void LogQueryInternal(ClientContextLock &lock, const string &query);

	unique_ptr<QueryResult> FetchResultInternal(ClientContextLock &lock, PendingQueryResult &pending);
	unique_ptr<DataChunk> FetchInternal(ClientContextLock &lock, Executor &executor, BaseQueryResult &result);

	unique_ptr<ClientContextLock> LockContext();

	void BeginTransactionInternal(ClientContextLock &lock, bool requires_valid_transaction);
	void BeginQueryInternal(ClientContextLock &lock, const string &query);
	PreservedError EndQueryInternal(ClientContextLock &lock, bool success, bool invalidate_transaction);

	PendingExecutionResult ExecuteTaskInternal(ClientContextLock &lock, PendingQueryResult &result);

	unique_ptr<PendingQueryResult> PendingStatementOrPreparedStatementInternal(
	    ClientContextLock &lock, const string &query, unique_ptr<SQLStatement> statement,
	    shared_ptr<PreparedStatementData> &prepared, PendingQueryParameters parameters);

	unique_ptr<PendingQueryResult> PendingQueryPreparedInternal(ClientContextLock &lock, const string &query,
	                                                            shared_ptr<PreparedStatementData> &prepared,
	                                                            PendingQueryParameters parameters);

	unique_ptr<PendingQueryResult> PendingQueryInternal(ClientContextLock &, const shared_ptr<Relation> &relation,
	                                                    bool allow_stream_result);

private:
	//! Lock on using the ClientContext in parallel
	mutex context_lock;
	//! The currently active query context
	unique_ptr<ActiveQueryContext> active_query;
	//! The current query progress
	atomic<double> query_progress;
};

class ClientContextLock {
public:
	explicit ClientContextLock(mutex &context_lock) : client_guard(context_lock) {
	}

	~ClientContextLock() {
	}

private:
	lock_guard<mutex> client_guard;
};

class ClientContextWrapper {
public:
	explicit ClientContextWrapper(const shared_ptr<ClientContext> &context)
	    : client_context(context) {

	      };
	shared_ptr<ClientContext> GetContext() {
		auto actual_context = client_context.lock();
		if (!actual_context) {
			throw ConnectionException("Connection has already been closed");
		}
		return actual_context;
	}

private:
	std::weak_ptr<ClientContext> client_context;
};

} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/statement/explain_statement.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/sql_statement.hpp
//
//
//===----------------------------------------------------------------------===//






//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/printer.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

enum class OutputStream : uint8_t { STREAM_STDOUT = 1, STREAM_STDERR = 2 };

//! Printer is a static class that allows printing to logs or stdout/stderr
class Printer {
public:
	//! Print the object to the stream
	DUCKDB_API static void Print(OutputStream stream, const string &str);
	//! Print the object to stderr
	DUCKDB_API static void Print(const string &str);
	//! Directly prints the string to stdout without a newline
	DUCKDB_API static void RawPrint(OutputStream stream, const string &str);
	//! Flush an output stream
	DUCKDB_API static void Flush(OutputStream stream);
	//! Whether or not we are printing to a terminal
	DUCKDB_API static bool IsTerminal(OutputStream stream);
	//! The terminal width
	DUCKDB_API static idx_t TerminalWidth();
};
} // namespace duckdb



namespace duckdb {

//! SQLStatement is the base class of any type of SQL statement.
class SQLStatement {
public:
	static constexpr const StatementType TYPE = StatementType::INVALID_STATEMENT;

public:
	explicit SQLStatement(StatementType type) : type(type) {
	}
	virtual ~SQLStatement() {
	}

	//! The statement type
	StatementType type;
	//! The statement location within the query string
	idx_t stmt_location = 0;
	//! The statement length within the query string
	idx_t stmt_length = 0;
	//! The number of prepared statement parameters (if any)
	idx_t n_param = 0;
	//! The map of named parameter to param index (if n_param and any named)
	case_insensitive_map_t<idx_t> named_param_map;
	//! The query text that corresponds to this SQL statement
	string query;

protected:
	SQLStatement(const SQLStatement &other) = default;

public:
	virtual string ToString() const {
		throw InternalException("ToString not supported for this type of SQLStatement: '%s'",
		                        StatementTypeToString(type));
	}
	//! Create a copy of this SelectStatement
	DUCKDB_API virtual unique_ptr<SQLStatement> Copy() const = 0;

public:
public:
	template <class TARGET>
	TARGET &Cast() {
		if (type != TARGET::TYPE && TARGET::TYPE != StatementType::INVALID_STATEMENT) {
			throw InternalException("Failed to cast statement to type - statement type mismatch");
		}
		return reinterpret_cast<TARGET &>(*this);
	}

	template <class TARGET>
	const TARGET &Cast() const {
		if (type != TARGET::TYPE && TARGET::TYPE != StatementType::INVALID_STATEMENT) {
			throw InternalException("Failed to cast statement to type - statement type mismatch");
		}
		return reinterpret_cast<const TARGET &>(*this);
	}
};
} // namespace duckdb


namespace duckdb {

enum class ExplainType : uint8_t { EXPLAIN_STANDARD, EXPLAIN_ANALYZE };

class ExplainStatement : public SQLStatement {
public:
	static constexpr const StatementType TYPE = StatementType::EXPLAIN_STATEMENT;

public:
	explicit ExplainStatement(unique_ptr<SQLStatement> stmt, ExplainType explain_type = ExplainType::EXPLAIN_STANDARD);

	unique_ptr<SQLStatement> stmt;
	ExplainType explain_type;

protected:
	ExplainStatement(const ExplainStatement &other);

public:
	unique_ptr<SQLStatement> Copy() const override;
};

} // namespace duckdb


#include <memory>

namespace duckdb {
struct BoundStatement;

class ClientContextWrapper;
class Binder;
class LogicalOperator;
class QueryNode;
class TableRef;

class Relation : public std::enable_shared_from_this<Relation> {
public:
	Relation(const std::shared_ptr<ClientContext> &context, RelationType type) : context(context), type(type) {
	}
	Relation(ClientContextWrapper &context, RelationType type) : context(context.GetContext()), type(type) {
	}
	virtual ~Relation() {
	}

	ClientContextWrapper context;

	RelationType type;

	shared_ptr<ExternalDependency> extra_dependencies;

public:
	DUCKDB_API virtual const vector<ColumnDefinition> &Columns() = 0;
	DUCKDB_API virtual unique_ptr<QueryNode> GetQueryNode();
	DUCKDB_API virtual BoundStatement Bind(Binder &binder);
	DUCKDB_API virtual string GetAlias();

	DUCKDB_API unique_ptr<QueryResult> ExecuteOrThrow();
	DUCKDB_API unique_ptr<QueryResult> Execute();
	DUCKDB_API string ToString();
	DUCKDB_API virtual string ToString(idx_t depth) = 0;

	DUCKDB_API void Print();
	DUCKDB_API void Head(idx_t limit = 10);

	DUCKDB_API shared_ptr<Relation> CreateView(const string &name, bool replace = true, bool temporary = false);
	DUCKDB_API shared_ptr<Relation> CreateView(const string &schema_name, const string &name, bool replace = true,
	                                           bool temporary = false);
	DUCKDB_API unique_ptr<QueryResult> Query(const string &sql);
	DUCKDB_API unique_ptr<QueryResult> Query(const string &name, const string &sql);

	//! Explain the query plan of this relation
	DUCKDB_API unique_ptr<QueryResult> Explain(ExplainType type = ExplainType::EXPLAIN_STANDARD);

	DUCKDB_API virtual unique_ptr<TableRef> GetTableRef();
	virtual bool IsReadOnly() {
		return true;
	}

public:
	// PROJECT
	DUCKDB_API shared_ptr<Relation> Project(const string &select_list);
	DUCKDB_API shared_ptr<Relation> Project(const string &expression, const string &alias);
	DUCKDB_API shared_ptr<Relation> Project(const string &select_list, const vector<string> &aliases);
	DUCKDB_API shared_ptr<Relation> Project(const vector<string> &expressions);
	DUCKDB_API shared_ptr<Relation> Project(const vector<string> &expressions, const vector<string> &aliases);

	// FILTER
	DUCKDB_API shared_ptr<Relation> Filter(const string &expression);
	DUCKDB_API shared_ptr<Relation> Filter(const vector<string> &expressions);

	// LIMIT
	DUCKDB_API shared_ptr<Relation> Limit(int64_t n, int64_t offset = 0);

	// ORDER
	DUCKDB_API shared_ptr<Relation> Order(const string &expression);
	DUCKDB_API shared_ptr<Relation> Order(const vector<string> &expressions);

	// JOIN operation
	DUCKDB_API shared_ptr<Relation> Join(const shared_ptr<Relation> &other, const string &condition,
	                                     JoinType type = JoinType::INNER);

	// CROSS PRODUCT operation
	DUCKDB_API shared_ptr<Relation> CrossProduct(const shared_ptr<Relation> &other);

	// SET operations
	DUCKDB_API shared_ptr<Relation> Union(const shared_ptr<Relation> &other);
	DUCKDB_API shared_ptr<Relation> Except(const shared_ptr<Relation> &other);
	DUCKDB_API shared_ptr<Relation> Intersect(const shared_ptr<Relation> &other);

	// DISTINCT operation
	DUCKDB_API shared_ptr<Relation> Distinct();

	// AGGREGATES
	DUCKDB_API shared_ptr<Relation> Aggregate(const string &aggregate_list);
	DUCKDB_API shared_ptr<Relation> Aggregate(const vector<string> &aggregates);
	DUCKDB_API shared_ptr<Relation> Aggregate(const string &aggregate_list, const string &group_list);
	DUCKDB_API shared_ptr<Relation> Aggregate(const vector<string> &aggregates, const vector<string> &groups);

	// ALIAS
	DUCKDB_API shared_ptr<Relation> Alias(const string &alias);

	//! Insert the data from this relation into a table
	DUCKDB_API shared_ptr<Relation> InsertRel(const string &schema_name, const string &table_name);
	DUCKDB_API void Insert(const string &table_name);
	DUCKDB_API void Insert(const string &schema_name, const string &table_name);
	//! Insert a row (i.e.,list of values) into a table
	DUCKDB_API void Insert(const vector<vector<Value>> &values);
	//! Create a table and insert the data from this relation into that table
	DUCKDB_API shared_ptr<Relation> CreateRel(const string &schema_name, const string &table_name);
	DUCKDB_API void Create(const string &table_name);
	DUCKDB_API void Create(const string &schema_name, const string &table_name);

	//! Write a relation to a CSV file
	DUCKDB_API shared_ptr<Relation>
	WriteCSVRel(const string &csv_file,
	            case_insensitive_map_t<vector<Value>> options = case_insensitive_map_t<vector<Value>>());
	DUCKDB_API void WriteCSV(const string &csv_file,
	                         case_insensitive_map_t<vector<Value>> options = case_insensitive_map_t<vector<Value>>());
	//! Write a relation to a Parquet file
	DUCKDB_API shared_ptr<Relation>
	WriteParquetRel(const string &parquet_file,
	                case_insensitive_map_t<vector<Value>> options = case_insensitive_map_t<vector<Value>>());
	DUCKDB_API void
	WriteParquet(const string &parquet_file,
	             case_insensitive_map_t<vector<Value>> options = case_insensitive_map_t<vector<Value>>());

	//! Update a table, can only be used on a TableRelation
	DUCKDB_API virtual void Update(const string &update, const string &condition = string());
	//! Delete from a table, can only be used on a TableRelation
	DUCKDB_API virtual void Delete(const string &condition = string());
	//! Create a relation from calling a table in/out function on the input relation
	//! Create a relation from calling a table in/out function on the input relation
	DUCKDB_API shared_ptr<Relation> TableFunction(const std::string &fname, const vector<Value> &values);
	DUCKDB_API shared_ptr<Relation> TableFunction(const std::string &fname, const vector<Value> &values,
	                                              const named_parameter_map_t &named_parameters);

public:
	//! Whether or not the relation inherits column bindings from its child or not, only relevant for binding
	virtual bool InheritsColumnBindings() {
		return false;
	}
	virtual Relation *ChildRelation() {
		return nullptr;
	}
	DUCKDB_API vector<shared_ptr<ExternalDependency>> GetAllDependencies();

protected:
	DUCKDB_API string RenderWhitespace(idx_t depth);

public:
	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}
	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
};

} // namespace duckdb





namespace duckdb {

class ColumnDataCollection;
class ClientContext;

class DatabaseInstance;
class DuckDB;
class LogicalOperator;
class SelectStatement;
struct BufferedCSVReaderOptions;

typedef void (*warning_callback)(std::string);

//! A connection to a database. This represents a (client) connection that can
//! be used to query the database.
class Connection {
public:
	DUCKDB_API explicit Connection(DuckDB &database);
	DUCKDB_API explicit Connection(DatabaseInstance &database);
	DUCKDB_API ~Connection();

	shared_ptr<ClientContext> context;
	warning_callback warning_cb;

public:
	//! Returns query profiling information for the current query
	DUCKDB_API string GetProfilingInformation(ProfilerPrintFormat format = ProfilerPrintFormat::QUERY_TREE);

	//! Interrupt execution of the current query
	DUCKDB_API void Interrupt();

	//! Enable query profiling
	DUCKDB_API void EnableProfiling();
	//! Disable query profiling
	DUCKDB_API void DisableProfiling();

	DUCKDB_API void SetWarningCallback(warning_callback);

	//! Enable aggressive verification/testing of queries, should only be used in testing
	DUCKDB_API void EnableQueryVerification();
	DUCKDB_API void DisableQueryVerification();
	//! Force parallel execution, even for smaller tables. Should only be used in testing.
	DUCKDB_API void ForceParallelism();

	//! Issues a query to the database and returns a QueryResult. This result can be either a StreamQueryResult or a
	//! MaterializedQueryResult. The result can be stepped through with calls to Fetch(). Note that there can only be
	//! one active StreamQueryResult per Connection object. Calling SendQuery() will invalidate any previously existing
	//! StreamQueryResult.
	DUCKDB_API unique_ptr<QueryResult> SendQuery(const string &query);
	//! Issues a query to the database and materializes the result (if necessary). Always returns a
	//! MaterializedQueryResult.
	DUCKDB_API unique_ptr<MaterializedQueryResult> Query(const string &query);
	//! Issues a query to the database and materializes the result (if necessary). Always returns a
	//! MaterializedQueryResult.
	DUCKDB_API unique_ptr<MaterializedQueryResult> Query(unique_ptr<SQLStatement> statement);
	// prepared statements
	template <typename... Args>
	unique_ptr<QueryResult> Query(const string &query, Args... args) {
		vector<Value> values;
		return QueryParamsRecursive(query, values, args...);
	}

	//! Issues a query to the database and returns a Pending Query Result. Note that "query" may only contain
	//! a single statement.
	DUCKDB_API unique_ptr<PendingQueryResult> PendingQuery(const string &query, bool allow_stream_result = false);
	//! Issues a query to the database and returns a Pending Query Result
	DUCKDB_API unique_ptr<PendingQueryResult> PendingQuery(unique_ptr<SQLStatement> statement,
	                                                       bool allow_stream_result = false);

	//! Prepare the specified query, returning a prepared statement object
	DUCKDB_API unique_ptr<PreparedStatement> Prepare(const string &query);
	//! Prepare the specified statement, returning a prepared statement object
	DUCKDB_API unique_ptr<PreparedStatement> Prepare(unique_ptr<SQLStatement> statement);

	//! Get the table info of a specific table (in the default schema), or nullptr if it cannot be found
	DUCKDB_API unique_ptr<TableDescription> TableInfo(const string &table_name);
	//! Get the table info of a specific table, or nullptr if it cannot be found
	DUCKDB_API unique_ptr<TableDescription> TableInfo(const string &schema_name, const string &table_name);

	//! Extract a set of SQL statements from a specific query
	DUCKDB_API vector<unique_ptr<SQLStatement>> ExtractStatements(const string &query);
	//! Extract the logical plan that corresponds to a query
	DUCKDB_API unique_ptr<LogicalOperator> ExtractPlan(const string &query);

	//! Appends a DataChunk to the specified table
	DUCKDB_API void Append(TableDescription &description, DataChunk &chunk);
	//! Appends a ColumnDataCollection to the specified table
	DUCKDB_API void Append(TableDescription &description, ColumnDataCollection &collection);

	//! Returns a relation that produces a table from this connection
	DUCKDB_API shared_ptr<Relation> Table(const string &tname);
	DUCKDB_API shared_ptr<Relation> Table(const string &schema_name, const string &table_name);
	//! Returns a relation that produces a view from this connection
	DUCKDB_API shared_ptr<Relation> View(const string &tname);
	DUCKDB_API shared_ptr<Relation> View(const string &schema_name, const string &table_name);
	//! Returns a relation that calls a specified table function
	DUCKDB_API shared_ptr<Relation> TableFunction(const string &tname);
	DUCKDB_API shared_ptr<Relation> TableFunction(const string &tname, const vector<Value> &values,
	                                              const named_parameter_map_t &named_parameters);
	DUCKDB_API shared_ptr<Relation> TableFunction(const string &tname, const vector<Value> &values);
	//! Returns a relation that produces values
	DUCKDB_API shared_ptr<Relation> Values(const vector<vector<Value>> &values);
	DUCKDB_API shared_ptr<Relation> Values(const vector<vector<Value>> &values, const vector<string> &column_names,
	                                       const string &alias = "values");
	DUCKDB_API shared_ptr<Relation> Values(const string &values);
	DUCKDB_API shared_ptr<Relation> Values(const string &values, const vector<string> &column_names,
	                                       const string &alias = "values");

	//! Reads CSV file
	DUCKDB_API shared_ptr<Relation> ReadCSV(const string &csv_file);
	DUCKDB_API shared_ptr<Relation> ReadCSV(const string &csv_file, BufferedCSVReaderOptions &options);
	DUCKDB_API shared_ptr<Relation> ReadCSV(const string &csv_file, const vector<string> &columns);

	//! Reads Parquet file
	DUCKDB_API shared_ptr<Relation> ReadParquet(const string &parquet_file, bool binary_as_string);
	//! Returns a relation from a query
	DUCKDB_API shared_ptr<Relation> RelationFromQuery(const string &query, const string &alias = "queryrelation",
	                                                  const string &error = "Expected a single SELECT statement");
	DUCKDB_API shared_ptr<Relation> RelationFromQuery(unique_ptr<SelectStatement> select_stmt,
	                                                  const string &alias = "queryrelation");

	//! Returns a substrait BLOB from a valid query
	DUCKDB_API string GetSubstrait(const string &query);
	//! Returns a Query Result from a substrait blob
	DUCKDB_API unique_ptr<QueryResult> FromSubstrait(const string &proto);
	//! Returns a substrait BLOB from a valid query
	DUCKDB_API string GetSubstraitJSON(const string &query);
	//! Returns a Query Result from a substrait JSON
	DUCKDB_API unique_ptr<QueryResult> FromSubstraitJSON(const string &json);
	DUCKDB_API void BeginTransaction();
	DUCKDB_API void Commit();
	DUCKDB_API void Rollback();
	DUCKDB_API void SetAutoCommit(bool auto_commit);
	DUCKDB_API bool IsAutoCommit();
	DUCKDB_API bool HasActiveTransaction();

	//! Fetch a list of table names that are required for a given query
	DUCKDB_API unordered_set<string> GetTableNames(const string &query);

	template <typename TR, typename... Args>
	void CreateScalarFunction(const string &name, TR (*udf_func)(Args...)) {
		scalar_function_t function = UDFWrapper::CreateScalarFunction<TR, Args...>(name, udf_func);
		UDFWrapper::RegisterFunction<TR, Args...>(name, function, *context);
	}

	template <typename TR, typename... Args>
	void CreateScalarFunction(const string &name, vector<LogicalType> args, LogicalType ret_type,
	                          TR (*udf_func)(Args...)) {
		scalar_function_t function =
		    UDFWrapper::CreateScalarFunction<TR, Args...>(name, args, std::move(ret_type), udf_func);
		UDFWrapper::RegisterFunction(name, args, ret_type, function, *context);
	}

	template <typename TR, typename... Args>
	void CreateVectorizedFunction(const string &name, scalar_function_t udf_func,
	                              LogicalType varargs = LogicalType::INVALID) {
		UDFWrapper::RegisterFunction<TR, Args...>(name, udf_func, *context, std::move(varargs));
	}

	void CreateVectorizedFunction(const string &name, vector<LogicalType> args, LogicalType ret_type,
	                              scalar_function_t udf_func, LogicalType varargs = LogicalType::INVALID) {
		UDFWrapper::RegisterFunction(name, std::move(args), std::move(ret_type), udf_func, *context,
		                             std::move(varargs));
	}

	//------------------------------------- Aggreate Functions ----------------------------------------//
	template <typename UDF_OP, typename STATE, typename TR, typename TA>
	void CreateAggregateFunction(const string &name) {
		AggregateFunction function = UDFWrapper::CreateAggregateFunction<UDF_OP, STATE, TR, TA>(name);
		UDFWrapper::RegisterAggrFunction(function, *context);
	}

	template <typename UDF_OP, typename STATE, typename TR, typename TA, typename TB>
	void CreateAggregateFunction(const string &name) {
		AggregateFunction function = UDFWrapper::CreateAggregateFunction<UDF_OP, STATE, TR, TA, TB>(name);
		UDFWrapper::RegisterAggrFunction(function, *context);
	}

	template <typename UDF_OP, typename STATE, typename TR, typename TA>
	void CreateAggregateFunction(const string &name, LogicalType ret_type, LogicalType input_typeA) {
		AggregateFunction function =
		    UDFWrapper::CreateAggregateFunction<UDF_OP, STATE, TR, TA>(name, ret_type, input_typeA);
		UDFWrapper::RegisterAggrFunction(function, *context);
	}

	template <typename UDF_OP, typename STATE, typename TR, typename TA, typename TB>
	void CreateAggregateFunction(const string &name, LogicalType ret_type, LogicalType input_typeA,
	                             LogicalType input_typeB) {
		AggregateFunction function =
		    UDFWrapper::CreateAggregateFunction<UDF_OP, STATE, TR, TA, TB>(name, ret_type, input_typeA, input_typeB);
		UDFWrapper::RegisterAggrFunction(function, *context);
	}

	void CreateAggregateFunction(const string &name, vector<LogicalType> arguments, LogicalType return_type,
	                             aggregate_size_t state_size, aggregate_initialize_t initialize,
	                             aggregate_update_t update, aggregate_combine_t combine, aggregate_finalize_t finalize,
	                             aggregate_simple_update_t simple_update = nullptr,
	                             bind_aggregate_function_t bind = nullptr,
	                             aggregate_destructor_t destructor = nullptr) {
		AggregateFunction function =
		    UDFWrapper::CreateAggregateFunction(name, arguments, return_type, state_size, initialize, update, combine,
		                                        finalize, simple_update, bind, destructor);
		UDFWrapper::RegisterAggrFunction(function, *context);
	}

private:
	unique_ptr<QueryResult> QueryParamsRecursive(const string &query, vector<Value> &values);

	template <typename T, typename... Args>
	unique_ptr<QueryResult> QueryParamsRecursive(const string &query, vector<Value> &values, T value, Args... args) {
		values.push_back(Value::CreateValue<T>(value));
		return QueryParamsRecursive(query, values, args...);
	}
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/main/database.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/main/config.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/access_mode.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

enum class AccessMode : uint8_t { UNDEFINED = 0, AUTOMATIC = 1, READ_ONLY = 2, READ_WRITE = 3 };

} // namespace duckdb





//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/optimizer_type.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

enum class OptimizerType : uint32_t {
	INVALID = 0,
	EXPRESSION_REWRITER,
	FILTER_PULLUP,
	FILTER_PUSHDOWN,
	REGEX_RANGE,
	IN_CLAUSE,
	JOIN_ORDER,
	DELIMINATOR,
	UNNEST_REWRITER,
	UNUSED_COLUMNS,
	STATISTICS_PROPAGATION,
	COMMON_SUBEXPRESSIONS,
	COMMON_AGGREGATE,
	COLUMN_LIFETIME,
	TOP_N,
	REORDER_FILTER,
	EXTENSION
};

string OptimizerTypeToString(OptimizerType type);
OptimizerType OptimizerTypeFromString(const string &str);

} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/set_scope.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

enum class SetScope : uint8_t {
	AUTOMATIC = 0,
	LOCAL = 1, /* unused */
	SESSION = 2,
	GLOBAL = 3
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/window_aggregation_mode.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

enum class WindowAggregationMode : uint32_t {
	//! Use the window aggregate API if available
	WINDOW = 0,
	//! Don't use window, but use combine if available
	COMBINE,
	//! Don't use combine or window (compute each frame separately)
	SEPARATE
};

} // namespace duckdb






//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/storage/compression/bitpacking.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

enum class BitpackingMode : uint8_t {
	AUTO,

	CONSTANT,
	CONSTANT_DELTA,
	DELTA_FOR,
	FOR
};

BitpackingMode BitpackingModeFromString(const string &str);
string BitpackingModeToString(const BitpackingMode &mode);

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/function/cast/default_casts.hpp
//
//
//===----------------------------------------------------------------------===//









namespace duckdb {

class CastFunctionSet;
struct FunctionLocalState;

//! Extra data that can be attached to a bind function of a cast, and is available during binding
struct BindCastInfo {
	DUCKDB_API virtual ~BindCastInfo();

	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}
	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
};

//! Extra data that can be returned by the bind of a cast, and is available during execution of a cast
struct BoundCastData {
	DUCKDB_API virtual ~BoundCastData();

	DUCKDB_API virtual unique_ptr<BoundCastData> Copy() const = 0;

	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}
	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
};

struct CastParameters {
	CastParameters() {
	}
	CastParameters(BoundCastData *cast_data, bool strict, string *error_message,
	               optional_ptr<FunctionLocalState> local_state)
	    : cast_data(cast_data), strict(strict), error_message(error_message), local_state(local_state) {
	}
	CastParameters(CastParameters &parent, optional_ptr<BoundCastData> cast_data,
	               optional_ptr<FunctionLocalState> local_state)
	    : cast_data(cast_data), strict(parent.strict), error_message(parent.error_message), local_state(local_state) {
	}

	//! The bound cast data (if any)
	optional_ptr<BoundCastData> cast_data;
	//! whether or not to enable strict casting
	bool strict = false;
	// out: error message in case cast has failed
	string *error_message = nullptr;
	//! Local state
	optional_ptr<FunctionLocalState> local_state;
};

struct CastLocalStateParameters {
	CastLocalStateParameters(optional_ptr<ClientContext> context_p, optional_ptr<BoundCastData> cast_data_p)
	    : context(context_p), cast_data(cast_data_p) {
	}
	CastLocalStateParameters(ClientContext &context_p, optional_ptr<BoundCastData> cast_data_p)
	    : context(&context_p), cast_data(cast_data_p) {
	}
	CastLocalStateParameters(CastLocalStateParameters &parent, optional_ptr<BoundCastData> cast_data_p)
	    : context(parent.context), cast_data(cast_data_p) {
	}

	optional_ptr<ClientContext> context;
	//! The bound cast data (if any)
	optional_ptr<BoundCastData> cast_data;
};

typedef bool (*cast_function_t)(Vector &source, Vector &result, idx_t count, CastParameters &parameters);
typedef unique_ptr<FunctionLocalState> (*init_cast_local_state_t)(CastLocalStateParameters &parameters);

struct BoundCastInfo {
	DUCKDB_API
	BoundCastInfo(
	    cast_function_t function, unique_ptr<BoundCastData> cast_data = nullptr,
	    init_cast_local_state_t init_local_state = nullptr); // NOLINT: allow explicit cast from cast_function_t
	cast_function_t function;
	init_cast_local_state_t init_local_state;
	unique_ptr<BoundCastData> cast_data;

public:
	BoundCastInfo Copy() const;
};

struct BindCastInput {
	DUCKDB_API BindCastInput(CastFunctionSet &function_set, optional_ptr<BindCastInfo> info,
	                         optional_ptr<ClientContext> context);

	CastFunctionSet &function_set;
	optional_ptr<BindCastInfo> info;
	optional_ptr<ClientContext> context;

public:
	DUCKDB_API BoundCastInfo GetCastFunction(const LogicalType &source, const LogicalType &target);
};

struct DefaultCasts {
	DUCKDB_API static BoundCastInfo GetDefaultCastFunction(BindCastInput &input, const LogicalType &source,
	                                                       const LogicalType &target);

	DUCKDB_API static bool NopCast(Vector &source, Vector &result, idx_t count, CastParameters &parameters);
	DUCKDB_API static bool TryVectorNullCast(Vector &source, Vector &result, idx_t count, CastParameters &parameters);
	DUCKDB_API static bool ReinterpretCast(Vector &source, Vector &result, idx_t count, CastParameters &parameters);

private:
	static BoundCastInfo BlobCastSwitch(BindCastInput &input, const LogicalType &source, const LogicalType &target);
	static BoundCastInfo BitCastSwitch(BindCastInput &input, const LogicalType &source, const LogicalType &target);
	static BoundCastInfo DateCastSwitch(BindCastInput &input, const LogicalType &source, const LogicalType &target);
	static BoundCastInfo DecimalCastSwitch(BindCastInput &input, const LogicalType &source, const LogicalType &target);
	static BoundCastInfo EnumCastSwitch(BindCastInput &input, const LogicalType &source, const LogicalType &target);
	static BoundCastInfo IntervalCastSwitch(BindCastInput &input, const LogicalType &source, const LogicalType &target);
	static BoundCastInfo ListCastSwitch(BindCastInput &input, const LogicalType &source, const LogicalType &target);
	static BoundCastInfo NumericCastSwitch(BindCastInput &input, const LogicalType &source, const LogicalType &target);
	static BoundCastInfo MapCastSwitch(BindCastInput &input, const LogicalType &source, const LogicalType &target);
	static BoundCastInfo PointerCastSwitch(BindCastInput &input, const LogicalType &source, const LogicalType &target);
	static BoundCastInfo StringCastSwitch(BindCastInput &input, const LogicalType &source, const LogicalType &target);
	static BoundCastInfo StructCastSwitch(BindCastInput &input, const LogicalType &source, const LogicalType &target);
	static BoundCastInfo TimeCastSwitch(BindCastInput &input, const LogicalType &source, const LogicalType &target);
	static BoundCastInfo TimeTzCastSwitch(BindCastInput &input, const LogicalType &source, const LogicalType &target);
	static BoundCastInfo TimestampCastSwitch(BindCastInput &input, const LogicalType &source,
	                                         const LogicalType &target);
	static BoundCastInfo TimestampTzCastSwitch(BindCastInput &input, const LogicalType &source,
	                                           const LogicalType &target);
	static BoundCastInfo TimestampNsCastSwitch(BindCastInput &input, const LogicalType &source,
	                                           const LogicalType &target);
	static BoundCastInfo TimestampMsCastSwitch(BindCastInput &input, const LogicalType &source,
	                                           const LogicalType &target);
	static BoundCastInfo TimestampSecCastSwitch(BindCastInput &input, const LogicalType &source,
	                                            const LogicalType &target);
	static BoundCastInfo UnionCastSwitch(BindCastInput &input, const LogicalType &source, const LogicalType &target);
	static BoundCastInfo UUIDCastSwitch(BindCastInput &input, const LogicalType &source, const LogicalType &target);

	static BoundCastInfo ImplicitToUnionCast(BindCastInput &input, const LogicalType &source,
	                                         const LogicalType &target);
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/function/replacement_scan.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

class ClientContext;
class TableRef;

struct ReplacementScanData {
	virtual ~ReplacementScanData() {
	}
};

typedef unique_ptr<TableRef> (*replacement_scan_t)(ClientContext &context, const string &table_name,
                                                   ReplacementScanData *data);

//! Replacement table scans are automatically attempted when a table name cannot be found in the schema
//! This allows you to do e.g. SELECT * FROM 'filename.csv', and automatically convert this into a CSV scan
struct ReplacementScan {
	explicit ReplacementScan(replacement_scan_t function, unique_ptr<ReplacementScanData> data_p = nullptr)
	    : function(function), data(std::move(data_p)) {
	}

	replacement_scan_t function;
	unique_ptr<ReplacementScanData> data;
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/optimizer/optimizer_extension.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {

//! The OptimizerExtensionInfo holds static information relevant to the optimizer extension
struct OptimizerExtensionInfo {
	virtual ~OptimizerExtensionInfo() {
	}
};

typedef void (*optimize_function_t)(ClientContext &context, OptimizerExtensionInfo *info,
                                    unique_ptr<LogicalOperator> &plan);

class OptimizerExtension {
public:
	//! The parse function of the parser extension.
	//! Takes a query string as input and returns ParserExtensionParseData (on success) or an error
	optimize_function_t optimize_function;

	//! Additional parser info passed to the parse function
	shared_ptr<OptimizerExtensionInfo> optimizer_info;
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/parser_extension.hpp
//
//
//===----------------------------------------------------------------------===//







namespace duckdb {

//! The ParserExtensionInfo holds static information relevant to the parser extension
//! It is made available in the parse_function, and will be kept alive as long as the database system is kept alive
struct ParserExtensionInfo {
	virtual ~ParserExtensionInfo() {
	}
};

//===--------------------------------------------------------------------===//
// Parse
//===--------------------------------------------------------------------===//
enum class ParserExtensionResultType : uint8_t { PARSE_SUCCESSFUL, DISPLAY_ORIGINAL_ERROR, DISPLAY_EXTENSION_ERROR };

//! The ParserExtensionParseData holds the result of a successful parse step
//! It will be passed along to the subsequent plan function
struct ParserExtensionParseData {
	virtual ~ParserExtensionParseData() {
	}

	virtual unique_ptr<ParserExtensionParseData> Copy() const = 0;
};

struct ParserExtensionParseResult {
	ParserExtensionParseResult() : type(ParserExtensionResultType::DISPLAY_ORIGINAL_ERROR) {
	}
	ParserExtensionParseResult(string error_p)
	    : type(ParserExtensionResultType::DISPLAY_EXTENSION_ERROR), error(std::move(error_p)) {
	}
	ParserExtensionParseResult(unique_ptr<ParserExtensionParseData> parse_data_p)
	    : type(ParserExtensionResultType::PARSE_SUCCESSFUL), parse_data(std::move(parse_data_p)) {
	}

	//! Whether or not parsing was successful
	ParserExtensionResultType type;
	//! The parse data (if successful)
	unique_ptr<ParserExtensionParseData> parse_data;
	//! The error message (if unsuccessful)
	string error;
};

typedef ParserExtensionParseResult (*parse_function_t)(ParserExtensionInfo *info, const string &query);
//===--------------------------------------------------------------------===//
// Plan
//===--------------------------------------------------------------------===//
struct ParserExtensionPlanResult {
	//! The table function to execute
	TableFunction function;
	//! Parameters to the function
	vector<Value> parameters;
	//! The set of databases that will be modified by this statement (empty for a read-only statement)
	unordered_set<string> modified_databases;
	//! Whether or not the statement requires a valid transaction to be executed
	bool requires_valid_transaction = true;
	//! What type of result set the statement returns
	StatementReturnType return_type = StatementReturnType::NOTHING;
};

typedef ParserExtensionPlanResult (*plan_function_t)(ParserExtensionInfo *info, ClientContext &context,
                                                     unique_ptr<ParserExtensionParseData> parse_data);

//===--------------------------------------------------------------------===//
// ParserExtension
//===--------------------------------------------------------------------===//
class ParserExtension {
public:
	//! The parse function of the parser extension.
	//! Takes a query string as input and returns ParserExtensionParseData (on success) or an error
	parse_function_t parse_function;

	//! The plan function of the parser extension
	//! Takes as input the result of the parse_function, and outputs various properties of the resulting plan
	plan_function_t plan_function;

	//! Additional parser info passed to the parse function
	shared_ptr<ParserExtensionInfo> parser_info;
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/planner/operator_extension.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/execution/physical_plan_generator.hpp
//
//
//===----------------------------------------------------------------------===//







//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/planner/operator/logical_limit_percent.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

//! LogicalLimitPercent represents a LIMIT PERCENT clause
class LogicalLimitPercent : public LogicalOperator {
public:
	static constexpr const LogicalOperatorType TYPE = LogicalOperatorType::LOGICAL_LIMIT_PERCENT;

public:
	LogicalLimitPercent(double limit_percent, int64_t offset_val, unique_ptr<Expression> limit,
	                    unique_ptr<Expression> offset)
	    : LogicalOperator(LogicalOperatorType::LOGICAL_LIMIT_PERCENT), limit_percent(limit_percent),
	      offset_val(offset_val), limit(std::move(limit)), offset(std::move(offset)) {
	}

	//! Limit percent and offset values in case they are constants, used in optimizations.
	double limit_percent;
	int64_t offset_val;
	//! The maximum amount of elements to emit
	unique_ptr<Expression> limit;
	//! The offset from the start to begin emitting elements
	unique_ptr<Expression> offset;

public:
	vector<ColumnBinding> GetColumnBindings() override {
		return children[0]->GetColumnBindings();
	}

	void Serialize(FieldWriter &writer) const override;
	static unique_ptr<LogicalOperator> Deserialize(LogicalDeserializationState &state, FieldReader &reader);
	idx_t EstimateCardinality(ClientContext &context) override;

protected:
	void ResolveTypes() override {
		types = children[0]->types;
	}
};
} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/catalog/dependency_list.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/catalog/catalog_entry_map.hpp
//
//
//===----------------------------------------------------------------------===//







namespace duckdb {
class CatalogEntry;

struct CatalogEntryHashFunction {
	uint64_t operator()(const reference<CatalogEntry> &a) const {
		std::hash<void *> hash_func;
		return hash_func((void *)&a.get());
	}
};

struct CatalogEntryEquality {
	bool operator()(const reference<CatalogEntry> &a, const reference<CatalogEntry> &b) const {
		return RefersToSameObject(a, b);
	}
};

using catalog_entry_set_t = unordered_set<reference<CatalogEntry>, CatalogEntryHashFunction, CatalogEntryEquality>;

template <typename T>
using catalog_entry_map_t = unordered_map<reference<CatalogEntry>, T, CatalogEntryHashFunction, CatalogEntryEquality>;

using catalog_entry_vector_t = vector<reference<CatalogEntry>>;

} // namespace duckdb


namespace duckdb {
class Catalog;
class CatalogEntry;

//! The DependencyList
class DependencyList {
	friend class DependencyManager;

public:
	DUCKDB_API void AddDependency(CatalogEntry &entry);

	DUCKDB_API void VerifyDependencies(Catalog &catalog, const string &name);

private:
	catalog_entry_set_t set;
};
} // namespace duckdb




namespace duckdb {
class ClientContext;
class ColumnDataCollection;

//! The physical plan generator generates a physical execution plan from a
//! logical query plan
class PhysicalPlanGenerator {
public:
	explicit PhysicalPlanGenerator(ClientContext &context);
	~PhysicalPlanGenerator();

	DependencyList dependencies;
	//! Recursive CTEs require at least one ChunkScan, referencing the working_table.
	//! This data structure is used to establish it.
	unordered_map<idx_t, std::shared_ptr<ColumnDataCollection>> recursive_cte_tables;

public:
	//! Creates a plan from the logical operator. This involves resolving column bindings and generating physical
	//! operator nodes.
	unique_ptr<PhysicalOperator> CreatePlan(unique_ptr<LogicalOperator> logical);

	//! Whether or not we can (or should) use a batch-index based operator for executing the given sink
	static bool UseBatchIndex(ClientContext &context, PhysicalOperator &plan);
	//! Whether or not we should preserve insertion order for executing the given sink
	static bool PreserveInsertionOrder(ClientContext &context, PhysicalOperator &plan);

protected:
	unique_ptr<PhysicalOperator> CreatePlan(LogicalOperator &op);

	unique_ptr<PhysicalOperator> CreatePlan(LogicalAggregate &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalAnyJoin &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalAsOfJoin &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalColumnDataGet &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalComparisonJoin &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalCreate &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalCreateTable &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalCreateIndex &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalCrossProduct &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalDelete &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalDelimGet &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalDelimJoin &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalDistinct &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalDummyScan &expr);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalEmptyResult &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalExpressionGet &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalExport &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalFilter &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalGet &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalLimit &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalLimitPercent &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalOrder &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalTopN &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalPositionalJoin &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalProjection &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalInsert &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalCopyToFile &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalExplain &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalSetOperation &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalUpdate &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalPrepare &expr);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalWindow &expr);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalExecute &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalPragma &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalSample &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalSet &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalReset &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalShow &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalSimple &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalUnnest &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalRecursiveCTE &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalCTERef &op);
	unique_ptr<PhysicalOperator> CreatePlan(LogicalPivot &op);

	unique_ptr<PhysicalOperator> ExtractAggregateExpressions(unique_ptr<PhysicalOperator> child,
	                                                         vector<unique_ptr<Expression>> &expressions,
	                                                         vector<unique_ptr<Expression>> &groups);

private:
	bool PreserveInsertionOrder(PhysicalOperator &plan);
	bool UseBatchIndex(PhysicalOperator &plan);

private:
	ClientContext &context;
};
} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/planner/binder.hpp
//
//
//===----------------------------------------------------------------------===//








//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/query_node.hpp
//
//
//===----------------------------------------------------------------------===//







//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/common_table_expression_info.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/statement/select_statement.hpp
//
//
//===----------------------------------------------------------------------===//






//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/tableref.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/enums/tableref_type.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

//===--------------------------------------------------------------------===//
// Table Reference Types
//===--------------------------------------------------------------------===//
enum class TableReferenceType : uint8_t {
	INVALID = 0,         // invalid table reference type
	BASE_TABLE = 1,      // base table reference
	SUBQUERY = 2,        // output of a subquery
	JOIN = 3,            // output of join
	TABLE_FUNCTION = 5,  // table producing function
	EXPRESSION_LIST = 6, // expression list
	CTE = 7,             // Recursive CTE
	EMPTY = 8,           // placeholder for empty FROM
	PIVOT = 9            // pivot statement
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/parsed_data/sample_options.hpp
//
//
//===----------------------------------------------------------------------===//








namespace duckdb {

enum class SampleMethod : uint8_t { SYSTEM_SAMPLE = 0, BERNOULLI_SAMPLE = 1, RESERVOIR_SAMPLE = 2 };

// **DEPRECATED**: Use EnumUtil directly instead.
string SampleMethodToString(SampleMethod method);

struct SampleOptions {
	Value sample_size;
	bool is_percentage;
	SampleMethod method;
	int64_t seed = -1;

	unique_ptr<SampleOptions> Copy();
	void Serialize(Serializer &serializer);
	static unique_ptr<SampleOptions> Deserialize(Deserializer &source);
	static bool Equals(SampleOptions *a, SampleOptions *b);
	void FormatSerialize(FormatSerializer &serializer) const;
	static unique_ptr<SampleOptions> FormatDeserialize(FormatDeserializer &deserializer);
};

} // namespace duckdb


namespace duckdb {
class Deserializer;
class Serializer;

//! Represents a generic expression that returns a table.
class TableRef {
public:
	static constexpr const TableReferenceType TYPE = TableReferenceType::INVALID;

public:
	explicit TableRef(TableReferenceType type) : type(type) {
	}
	virtual ~TableRef() {
	}

	TableReferenceType type;
	string alias;
	//! Sample options (if any)
	unique_ptr<SampleOptions> sample;
	//! The location in the query (if any)
	idx_t query_location = DConstants::INVALID_INDEX;

public:
	//! Convert the object to a string
	virtual string ToString() const = 0;
	string BaseToString(string result) const;
	string BaseToString(string result, const vector<string> &column_name_alias) const;
	void Print();

	virtual bool Equals(const TableRef &other) const;
	static bool Equals(const unique_ptr<TableRef> &left, const unique_ptr<TableRef> &right);

	virtual unique_ptr<TableRef> Copy() = 0;

	//! Serializes a TableRef to a stand-alone binary blob
	DUCKDB_API void Serialize(Serializer &serializer) const;
	//! Serializes a TableRef to a stand-alone binary blob
	DUCKDB_API virtual void Serialize(FieldWriter &writer) const = 0;
	//! Deserializes a blob back into a TableRef
	DUCKDB_API static unique_ptr<TableRef> Deserialize(Deserializer &source);
	//! Copy the properties of this table ref to the target
	void CopyProperties(TableRef &target) const;

	virtual void FormatSerialize(FormatSerializer &serializer) const;
	static unique_ptr<TableRef> FormatDeserialize(FormatDeserializer &deserializer);

public:
	template <class TARGET>
	TARGET &Cast() {
		if (type != TARGET::TYPE && TARGET::TYPE != TableReferenceType::INVALID) {
			throw InternalException("Failed to cast constraint to type - constraint type mismatch");
		}
		return reinterpret_cast<TARGET &>(*this);
	}

	template <class TARGET>
	const TARGET &Cast() const {
		if (type != TARGET::TYPE && TARGET::TYPE != TableReferenceType::INVALID) {
			throw InternalException("Failed to cast constraint to type - constraint type mismatch");
		}
		return reinterpret_cast<const TARGET &>(*this);
	}
};
} // namespace duckdb



namespace duckdb {

class QueryNode;
class FormatSerializer;
class FormatDeserializer;

//! SelectStatement is a typical SELECT clause
class SelectStatement : public SQLStatement {
public:
	static constexpr const StatementType TYPE = StatementType::SELECT_STATEMENT;

public:
	SelectStatement() : SQLStatement(StatementType::SELECT_STATEMENT) {
	}

	//! The main query node
	unique_ptr<QueryNode> node;

protected:
	SelectStatement(const SelectStatement &other);

public:
	//! Convert the SELECT statement to a string
	DUCKDB_API string ToString() const override;
	//! Create a copy of this SelectStatement
	DUCKDB_API unique_ptr<SQLStatement> Copy() const override;
	//! Serializes a SelectStatement to a stand-alone binary blob
	void Serialize(Serializer &serializer) const;
	//! Deserializes a blob back into a SelectStatement, returns nullptr if
	//! deserialization is not possible
	static unique_ptr<SelectStatement> Deserialize(Deserializer &source);
	//! Whether or not the statements are equivalent
	bool Equals(const SQLStatement &other) const;

	void FormatSerialize(FormatSerializer &serializer) const;
	static unique_ptr<SelectStatement> FormatDeserialize(FormatDeserializer &deserializer);
};
} // namespace duckdb


namespace duckdb {

class SelectStatement;

struct CommonTableExpressionInfo {
	vector<string> aliases;
	unique_ptr<SelectStatement> query;

	void FormatSerialize(FormatSerializer &serializer) const;
	static unique_ptr<CommonTableExpressionInfo> FormatDeserialize(FormatDeserializer &deserializer);
	unique_ptr<CommonTableExpressionInfo> Copy();
};

} // namespace duckdb




namespace duckdb {

class FormatDeserializer;
class FormatSerializer;

enum class QueryNodeType : uint8_t {
	SELECT_NODE = 1,
	SET_OPERATION_NODE = 2,
	BOUND_SUBQUERY_NODE = 3,
	RECURSIVE_CTE_NODE = 4
};

struct CommonTableExpressionInfo;

class CommonTableExpressionMap {
public:
	CommonTableExpressionMap();

	case_insensitive_map_t<unique_ptr<CommonTableExpressionInfo>> map;

public:
	string ToString() const;
	CommonTableExpressionMap Copy() const;

	void FormatSerialize(FormatSerializer &serializer) const;
	// static void FormatDeserialize(FormatDeserializer &deserializer, CommonTableExpressionMap &ret);
	static CommonTableExpressionMap FormatDeserialize(FormatDeserializer &deserializer);
};

class QueryNode {
public:
	explicit QueryNode(QueryNodeType type) : type(type) {
	}
	virtual ~QueryNode() {
	}

	//! The type of the query node, either SetOperation or Select
	QueryNodeType type;
	//! The set of result modifiers associated with this query node
	vector<unique_ptr<ResultModifier>> modifiers;
	//! CTEs (used by SelectNode and SetOperationNode)
	CommonTableExpressionMap cte_map;

	virtual const vector<unique_ptr<ParsedExpression>> &GetSelectList() const = 0;

public:
	//! Convert the query node to a string
	virtual string ToString() const = 0;

	virtual bool Equals(const QueryNode *other) const;

	//! Create a copy of this QueryNode
	virtual unique_ptr<QueryNode> Copy() const = 0;
	//! Serializes a QueryNode to a stand-alone binary blob
	DUCKDB_API void Serialize(Serializer &serializer) const;
	//! Serializes a QueryNode to a stand-alone binary blob
	DUCKDB_API virtual void Serialize(FieldWriter &writer) const = 0;
	//! Deserializes a blob back into a QueryNode
	DUCKDB_API static unique_ptr<QueryNode> Deserialize(Deserializer &source);

	string ResultModifiersToString() const;

	//! Adds a distinct modifier to the query node
	void AddDistinct();

	virtual void FormatSerialize(FormatSerializer &serializer) const;
	static unique_ptr<QueryNode> FormatDeserialize(FormatDeserializer &deserializer);

protected:
	//! Copy base QueryNode properties from another expression to this one,
	//! used in Copy method
	void CopyProperties(QueryNode &other) const;

public:
	template <class TARGET>
	TARGET &Cast() {
		if (type != TARGET::TYPE) {
			throw InternalException("Failed to cast query node to type - query node type mismatch");
		}
		return reinterpret_cast<TARGET &>(*this);
	}

	template <class TARGET>
	const TARGET &Cast() const {
		if (type != TARGET::TYPE) {
			throw InternalException("Failed to cast query node to type - query node type mismatch");
		}
		return reinterpret_cast<const TARGET &>(*this);
	}
};

} // namespace duckdb






//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/planner/expression/bound_columnref_expression.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {

class FieldReader;
class FieldWriter;

//! A BoundColumnRef expression represents a ColumnRef expression that was bound to an actual table and column index. It
//! is not yet executable, however. The ColumnBindingResolver transforms the BoundColumnRefExpressions into
//! BoundExpressions, which refer to indexes into the physical chunks that pass through the executor.
class BoundColumnRefExpression : public Expression {
public:
	static constexpr const ExpressionClass TYPE = ExpressionClass::BOUND_COLUMN_REF;

public:
	BoundColumnRefExpression(LogicalType type, ColumnBinding binding, idx_t depth = 0);
	BoundColumnRefExpression(string alias, LogicalType type, ColumnBinding binding, idx_t depth = 0);

	//! Column index set by the binder, used to generate the final BoundExpression
	ColumnBinding binding;
	//! The subquery depth (i.e. depth 0 = current query, depth 1 = parent query, depth 2 = parent of parent, etc...).
	//! This is only non-zero for correlated expressions inside subqueries.
	idx_t depth;

public:
	bool IsScalar() const override {
		return false;
	}
	bool IsFoldable() const override {
		return false;
	}

	string ToString() const override;
	string GetName() const override;

	bool Equals(const BaseExpression &other) const override;
	hash_t Hash() const override;

	unique_ptr<Expression> Copy() override;

	void Serialize(FieldWriter &writer) const override;
	static unique_ptr<Expression> Deserialize(ExpressionDeserializationState &state, FieldReader &reader);
};
} // namespace duckdb




namespace duckdb {
class BoundResultModifier;
class BoundSelectNode;
class ClientContext;
class ExpressionBinder;
class LimitModifier;
class OrderBinder;
class TableCatalogEntry;
class ViewCatalogEntry;
class TableMacroCatalogEntry;
class UpdateSetInfo;
class LogicalProjection;

class ColumnList;
class ExternalDependency;
class TableFunction;
class TableStorageInfo;

struct CreateInfo;
struct BoundCreateTableInfo;
struct BoundCreateFunctionInfo;
struct CommonTableExpressionInfo;
struct BoundParameterMap;

enum class BindingMode : uint8_t { STANDARD_BINDING, EXTRACT_NAMES };

struct CorrelatedColumnInfo {
	ColumnBinding binding;
	LogicalType type;
	string name;
	idx_t depth;

	CorrelatedColumnInfo(ColumnBinding binding, LogicalType type_p, string name_p, idx_t depth)
	    : binding(binding), type(std::move(type_p)), name(std::move(name_p)), depth(depth) {
	}
	explicit CorrelatedColumnInfo(BoundColumnRefExpression &expr)
	    : CorrelatedColumnInfo(expr.binding, expr.return_type, expr.GetName(), expr.depth) {
	}

	bool operator==(const CorrelatedColumnInfo &rhs) const {
		return binding == rhs.binding;
	}
};

//! Bind the parsed query tree to the actual columns present in the catalog.
/*!
  The binder is responsible for binding tables and columns to actual physical
  tables and columns in the catalog. In the process, it also resolves types of
  all expressions.
*/
class Binder : public std::enable_shared_from_this<Binder> {
	friend class ExpressionBinder;
	friend class RecursiveSubqueryPlanner;

public:
	DUCKDB_API static shared_ptr<Binder> CreateBinder(ClientContext &context, optional_ptr<Binder> parent = nullptr,
	                                                  bool inherit_ctes = true);

	//! The client context
	ClientContext &context;
	//! A mapping of names to common table expressions
	case_insensitive_map_t<reference<CommonTableExpressionInfo>> CTE_bindings; // NOLINT
	//! The CTEs that have already been bound
	reference_set_t<CommonTableExpressionInfo> bound_ctes;
	//! The bind context
	BindContext bind_context;
	//! The set of correlated columns bound by this binder (FIXME: this should probably be an unordered_set and not a
	//! vector)
	vector<CorrelatedColumnInfo> correlated_columns;
	//! The set of parameter expressions bound by this binder
	optional_ptr<BoundParameterMap> parameters;
	//! Statement properties
	StatementProperties properties;
	//! The alias for the currently processing subquery, if it exists
	string alias;
	//! Macro parameter bindings (if any)
	optional_ptr<DummyBinding> macro_binding;
	//! The intermediate lambda bindings to bind nested lambdas (if any)
	optional_ptr<vector<DummyBinding>> lambda_bindings;

public:
	DUCKDB_API BoundStatement Bind(SQLStatement &statement);
	DUCKDB_API BoundStatement Bind(QueryNode &node);

	unique_ptr<BoundCreateTableInfo> BindCreateTableInfo(unique_ptr<CreateInfo> info);
	unique_ptr<BoundCreateTableInfo> BindCreateTableInfo(unique_ptr<CreateInfo> info, SchemaCatalogEntry &schema);

	vector<unique_ptr<Expression>> BindCreateIndexExpressions(TableCatalogEntry &table, CreateIndexInfo &info);

	void BindCreateViewInfo(CreateViewInfo &base);
	SchemaCatalogEntry &BindSchema(CreateInfo &info);
	SchemaCatalogEntry &BindCreateFunctionInfo(CreateInfo &info);

	//! Check usage, and cast named parameters to their types
	static void BindNamedParameters(named_parameter_type_map_t &types, named_parameter_map_t &values,
	                                QueryErrorContext &error_context, string &func_name);

	unique_ptr<BoundTableRef> Bind(TableRef &ref);
	unique_ptr<LogicalOperator> CreatePlan(BoundTableRef &ref);

	//! Generates an unused index for a table
	idx_t GenerateTableIndex();

	//! Add a common table expression to the binder
	void AddCTE(const string &name, CommonTableExpressionInfo &cte);
	//! Find a common table expression by name; returns nullptr if none exists
	optional_ptr<CommonTableExpressionInfo> FindCTE(const string &name, bool skip = false);

	bool CTEIsAlreadyBound(CommonTableExpressionInfo &cte);

	//! Add the view to the set of currently bound views - used for detecting recursive view definitions
	void AddBoundView(ViewCatalogEntry &view);

	void PushExpressionBinder(ExpressionBinder &binder);
	void PopExpressionBinder();
	void SetActiveBinder(ExpressionBinder &binder);
	ExpressionBinder &GetActiveBinder();
	bool HasActiveBinder();

	vector<reference<ExpressionBinder>> &GetActiveBinders();

	void MergeCorrelatedColumns(vector<CorrelatedColumnInfo> &other);
	//! Add a correlated column to this binder (if it does not exist)
	void AddCorrelatedColumn(const CorrelatedColumnInfo &info);

	string FormatError(ParsedExpression &expr_context, const string &message);
	string FormatError(TableRef &ref_context, const string &message);

	string FormatErrorRecursive(idx_t query_location, const string &message, vector<ExceptionFormatValue> &values);
	template <class T, typename... ARGS>
	string FormatErrorRecursive(idx_t query_location, const string &msg, vector<ExceptionFormatValue> &values, T param,
	                            ARGS... params) {
		values.push_back(ExceptionFormatValue::CreateFormatValue<T>(param));
		return FormatErrorRecursive(query_location, msg, values, params...);
	}

	template <typename... ARGS>
	string FormatError(idx_t query_location, const string &msg, ARGS... params) {
		vector<ExceptionFormatValue> values;
		return FormatErrorRecursive(query_location, msg, values, params...);
	}

	unique_ptr<LogicalOperator> BindUpdateSet(LogicalOperator &op, unique_ptr<LogicalOperator> root,
	                                          UpdateSetInfo &set_info, TableCatalogEntry &table,
	                                          vector<PhysicalIndex> &columns);
	void BindDoUpdateSetExpressions(const string &table_alias, LogicalInsert &insert, UpdateSetInfo &set_info,
	                                TableCatalogEntry &table, TableStorageInfo &storage_info);
	void BindOnConflictClause(LogicalInsert &insert, TableCatalogEntry &table, InsertStatement &stmt);

	static void BindSchemaOrCatalog(ClientContext &context, string &catalog, string &schema);
	static void BindLogicalType(ClientContext &context, LogicalType &type, optional_ptr<Catalog> catalog = nullptr,
	                            const string &schema = INVALID_SCHEMA);

	bool HasMatchingBinding(const string &table_name, const string &column_name, string &error_message);
	bool HasMatchingBinding(const string &schema_name, const string &table_name, const string &column_name,
	                        string &error_message);
	bool HasMatchingBinding(const string &catalog_name, const string &schema_name, const string &table_name,
	                        const string &column_name, string &error_message);

	void SetBindingMode(BindingMode mode);
	BindingMode GetBindingMode();
	void AddTableName(string table_name);
	const unordered_set<string> &GetTableNames();
	optional_ptr<SQLStatement> GetRootStatement() {
		return root_statement;
	}

	void SetCanContainNulls(bool can_contain_nulls);

private:
	//! The parent binder (if any)
	shared_ptr<Binder> parent;
	//! The vector of active binders
	vector<reference<ExpressionBinder>> active_binders;
	//! The count of bound_tables
	idx_t bound_tables;
	//! Whether or not the binder has any unplanned subqueries that still need to be planned
	bool has_unplanned_subqueries = false;
	//! Whether or not subqueries should be planned already
	bool plan_subquery = true;
	//! Whether CTEs should reference the parent binder (if it exists)
	bool inherit_ctes = true;
	//! Whether or not the binder can contain NULLs as the root of expressions
	bool can_contain_nulls = false;
	//! The root statement of the query that is currently being parsed
	optional_ptr<SQLStatement> root_statement;
	//! Binding mode
	BindingMode mode = BindingMode::STANDARD_BINDING;
	//! Table names extracted for BindingMode::EXTRACT_NAMES
	unordered_set<string> table_names;
	//! The set of bound views
	reference_set_t<ViewCatalogEntry> bound_views;

private:
	//! Bind the expressions of generated columns to check for errors
	void BindGeneratedColumns(BoundCreateTableInfo &info);
	//! Bind the default values of the columns of a table
	void BindDefaultValues(const ColumnList &columns, vector<unique_ptr<Expression>> &bound_defaults);
	//! Bind a limit value (LIMIT or OFFSET)
	unique_ptr<Expression> BindDelimiter(ClientContext &context, OrderBinder &order_binder,
	                                     unique_ptr<ParsedExpression> delimiter, const LogicalType &type,
	                                     Value &delimiter_value);

	//! Move correlated expressions from the child binder to this binder
	void MoveCorrelatedExpressions(Binder &other);

	BoundStatement Bind(SelectStatement &stmt);
	BoundStatement Bind(InsertStatement &stmt);
	BoundStatement Bind(CopyStatement &stmt);
	BoundStatement Bind(DeleteStatement &stmt);
	BoundStatement Bind(UpdateStatement &stmt);
	BoundStatement Bind(CreateStatement &stmt);
	BoundStatement Bind(DropStatement &stmt);
	BoundStatement Bind(AlterStatement &stmt);
	BoundStatement Bind(PrepareStatement &stmt);
	BoundStatement Bind(ExecuteStatement &stmt);
	BoundStatement Bind(TransactionStatement &stmt);
	BoundStatement Bind(PragmaStatement &stmt);
	BoundStatement Bind(ExplainStatement &stmt);
	BoundStatement Bind(VacuumStatement &stmt);
	BoundStatement Bind(RelationStatement &stmt);
	BoundStatement Bind(ShowStatement &stmt);
	BoundStatement Bind(CallStatement &stmt);
	BoundStatement Bind(ExportStatement &stmt);
	BoundStatement Bind(ExtensionStatement &stmt);
	BoundStatement Bind(SetStatement &stmt);
	BoundStatement Bind(SetVariableStatement &stmt);
	BoundStatement Bind(ResetVariableStatement &stmt);
	BoundStatement Bind(LoadStatement &stmt);
	BoundStatement Bind(LogicalPlanStatement &stmt);
	BoundStatement Bind(AttachStatement &stmt);
	BoundStatement Bind(DetachStatement &stmt);

	BoundStatement BindReturning(vector<unique_ptr<ParsedExpression>> returning_list, TableCatalogEntry &table,
	                             const string &alias, idx_t update_table_index,
	                             unique_ptr<LogicalOperator> child_operator, BoundStatement result);

	unique_ptr<QueryNode> BindTableMacro(FunctionExpression &function, TableMacroCatalogEntry &macro_func, idx_t depth);

	unique_ptr<BoundQueryNode> BindNode(SelectNode &node);
	unique_ptr<BoundQueryNode> BindNode(SetOperationNode &node);
	unique_ptr<BoundQueryNode> BindNode(RecursiveCTENode &node);
	unique_ptr<BoundQueryNode> BindNode(QueryNode &node);

	unique_ptr<LogicalOperator> VisitQueryNode(BoundQueryNode &node, unique_ptr<LogicalOperator> root);
	unique_ptr<LogicalOperator> CreatePlan(BoundRecursiveCTENode &node);
	unique_ptr<LogicalOperator> CreatePlan(BoundSelectNode &statement);
	unique_ptr<LogicalOperator> CreatePlan(BoundSetOperationNode &node);
	unique_ptr<LogicalOperator> CreatePlan(BoundQueryNode &node);

	unique_ptr<BoundTableRef> Bind(BaseTableRef &ref);
	unique_ptr<BoundTableRef> Bind(JoinRef &ref);
	unique_ptr<BoundTableRef> Bind(SubqueryRef &ref, optional_ptr<CommonTableExpressionInfo> cte = nullptr);
	unique_ptr<BoundTableRef> Bind(TableFunctionRef &ref);
	unique_ptr<BoundTableRef> Bind(EmptyTableRef &ref);
	unique_ptr<BoundTableRef> Bind(ExpressionListRef &ref);
	unique_ptr<BoundTableRef> Bind(PivotRef &expr);

	unique_ptr<SelectNode> BindPivot(PivotRef &expr, vector<unique_ptr<ParsedExpression>> all_columns);
	unique_ptr<SelectNode> BindUnpivot(Binder &child_binder, PivotRef &expr,
	                                   vector<unique_ptr<ParsedExpression>> all_columns,
	                                   unique_ptr<ParsedExpression> &where_clause);
	unique_ptr<BoundTableRef> BindBoundPivot(PivotRef &expr);

	bool BindTableFunctionParameters(TableFunctionCatalogEntry &table_function,
	                                 vector<unique_ptr<ParsedExpression>> &expressions, vector<LogicalType> &arguments,
	                                 vector<Value> &parameters, named_parameter_map_t &named_parameters,
	                                 unique_ptr<BoundSubqueryRef> &subquery, string &error);
	bool BindTableInTableOutFunction(vector<unique_ptr<ParsedExpression>> &expressions,
	                                 unique_ptr<BoundSubqueryRef> &subquery, string &error);
	unique_ptr<LogicalOperator> BindTableFunction(TableFunction &function, vector<Value> parameters);
	unique_ptr<LogicalOperator>
	BindTableFunctionInternal(TableFunction &table_function, const string &function_name, vector<Value> parameters,
	                          named_parameter_map_t named_parameters, vector<LogicalType> input_table_types,
	                          vector<string> input_table_names, const vector<string> &column_name_alias,
	                          unique_ptr<ExternalDependency> external_dependency);

	unique_ptr<LogicalOperator> CreatePlan(BoundBaseTableRef &ref);
	unique_ptr<LogicalOperator> CreatePlan(BoundJoinRef &ref);
	unique_ptr<LogicalOperator> CreatePlan(BoundSubqueryRef &ref);
	unique_ptr<LogicalOperator> CreatePlan(BoundTableFunction &ref);
	unique_ptr<LogicalOperator> CreatePlan(BoundEmptyTableRef &ref);
	unique_ptr<LogicalOperator> CreatePlan(BoundExpressionListRef &ref);
	unique_ptr<LogicalOperator> CreatePlan(BoundCTERef &ref);
	unique_ptr<LogicalOperator> CreatePlan(BoundPivotRef &ref);

	BoundStatement BindCopyTo(CopyStatement &stmt);
	BoundStatement BindCopyFrom(CopyStatement &stmt);

	void BindModifiers(OrderBinder &order_binder, QueryNode &statement, BoundQueryNode &result);
	void BindModifierTypes(BoundQueryNode &result, const vector<LogicalType> &sql_types, idx_t projection_index);

	BoundStatement BindSummarize(ShowStatement &stmt);
	unique_ptr<BoundResultModifier> BindLimit(OrderBinder &order_binder, LimitModifier &limit_mod);
	unique_ptr<BoundResultModifier> BindLimitPercent(OrderBinder &order_binder, LimitPercentModifier &limit_mod);
	unique_ptr<Expression> BindOrderExpression(OrderBinder &order_binder, unique_ptr<ParsedExpression> expr);

	unique_ptr<LogicalOperator> PlanFilter(unique_ptr<Expression> condition, unique_ptr<LogicalOperator> root);

	void PlanSubqueries(unique_ptr<Expression> &expr, unique_ptr<LogicalOperator> &root);
	unique_ptr<Expression> PlanSubquery(BoundSubqueryExpression &expr, unique_ptr<LogicalOperator> &root);
	unique_ptr<LogicalOperator> PlanLateralJoin(unique_ptr<LogicalOperator> left, unique_ptr<LogicalOperator> right,
	                                            vector<CorrelatedColumnInfo> &correlated_columns,
	                                            JoinType join_type = JoinType::INNER,
	                                            unique_ptr<Expression> condition = nullptr);

	unique_ptr<LogicalOperator> CastLogicalOperatorToTypes(vector<LogicalType> &source_types,
	                                                       vector<LogicalType> &target_types,
	                                                       unique_ptr<LogicalOperator> op);

	string FindBinding(const string &using_column, const string &join_side);
	bool TryFindBinding(const string &using_column, const string &join_side, string &result);

	void AddUsingBindingSet(unique_ptr<UsingColumnSet> set);
	string RetrieveUsingBinding(Binder &current_binder, optional_ptr<UsingColumnSet> current_set,
	                            const string &column_name, const string &join_side);

	void AddCTEMap(CommonTableExpressionMap &cte_map);

	void ExpandStarExpressions(vector<unique_ptr<ParsedExpression>> &select_list,
	                           vector<unique_ptr<ParsedExpression>> &new_select_list);
	void ExpandStarExpression(unique_ptr<ParsedExpression> expr, vector<unique_ptr<ParsedExpression>> &new_select_list);
	bool FindStarExpression(unique_ptr<ParsedExpression> &expr, StarExpression **star, bool is_root, bool in_columns);
	void ReplaceStarExpression(unique_ptr<ParsedExpression> &expr, unique_ptr<ParsedExpression> &replacement);
	void BindWhereStarExpression(unique_ptr<ParsedExpression> &expr);

	//! If only a schema name is provided (e.g. "a.b") then figure out if "a" is a schema or a catalog name
	void BindSchemaOrCatalog(string &catalog_name, string &schema_name);
	SchemaCatalogEntry &BindCreateSchema(CreateInfo &info);

	unique_ptr<BoundQueryNode> BindSelectNode(SelectNode &statement, unique_ptr<BoundTableRef> from_table);

public:
	// This should really be a private constructor, but make_shared does not allow it...
	// If you are thinking about calling this, you should probably call Binder::CreateBinder
	Binder(bool i_know_what_i_am_doing, ClientContext &context, shared_ptr<Binder> parent, bool inherit_ctes);
};

} // namespace duckdb


namespace duckdb {

//! The OperatorExtensionInfo holds static information relevant to the operator extension
struct OperatorExtensionInfo {
	virtual ~OperatorExtensionInfo() {
	}
};

typedef BoundStatement (*bind_function_t)(ClientContext &context, Binder &binder, OperatorExtensionInfo *info,
                                          SQLStatement &statement);

// forward declaration to avoid circular reference
struct LogicalExtensionOperator;

class OperatorExtension {
public:
	bind_function_t Bind;

	//! Additional info passed to the CreatePlan & Bind functions
	shared_ptr<OperatorExtensionInfo> operator_info;

	virtual std::string GetName() = 0;
	virtual unique_ptr<LogicalExtensionOperator> Deserialize(LogicalDeserializationState &state,
	                                                         FieldReader &reader) = 0;

	virtual ~OperatorExtension() {
	}
};

} // namespace duckdb



namespace duckdb {
class BufferPool;
class CastFunctionSet;
class ClientContext;
class ErrorManager;
class CompressionFunction;
class TableFunctionRef;
class OperatorExtension;
class StorageExtension;

struct CompressionFunctionSet;
struct DBConfig;

enum class CheckpointAbort : uint8_t {
	NO_ABORT = 0,
	DEBUG_ABORT_BEFORE_TRUNCATE = 1,
	DEBUG_ABORT_BEFORE_HEADER = 2,
	DEBUG_ABORT_AFTER_FREE_LIST_WRITE = 3
};

typedef void (*set_global_function_t)(DatabaseInstance *db, DBConfig &config, const Value &parameter);
typedef void (*set_local_function_t)(ClientContext &context, const Value &parameter);
typedef void (*reset_global_function_t)(DatabaseInstance *db, DBConfig &config);
typedef void (*reset_local_function_t)(ClientContext &context);
typedef Value (*get_setting_function_t)(ClientContext &context);

struct ConfigurationOption {
	const char *name;
	const char *description;
	LogicalTypeId parameter_type;
	set_global_function_t set_global;
	set_local_function_t set_local;
	reset_global_function_t reset_global;
	reset_local_function_t reset_local;
	get_setting_function_t get_setting;
};

typedef void (*set_option_callback_t)(ClientContext &context, SetScope scope, Value &parameter);

struct ExtensionOption {
	ExtensionOption(string description_p, LogicalType type_p, set_option_callback_t set_function_p,
	                Value default_value_p)
	    : description(std::move(description_p)), type(std::move(type_p)), set_function(set_function_p),
	      default_value(std::move(default_value_p)) {
	}

	string description;
	LogicalType type;
	set_option_callback_t set_function;
	Value default_value;
};

struct DBConfigOptions {
	//! Database file path. May be empty for in-memory mode
	string database_path;
	//! Database type. If empty, automatically extracted from `database_path`, where a `type:path` syntax is expected
	string database_type;
	//! Access mode of the database (AUTOMATIC, READ_ONLY or READ_WRITE)
	AccessMode access_mode = AccessMode::AUTOMATIC;
	//! Checkpoint when WAL reaches this size (default: 16MB)
	idx_t checkpoint_wal_size = 1 << 24;
	//! Whether or not to use Direct IO, bypassing operating system buffers
	bool use_direct_io = false;
	//! Whether extensions should be loaded on start-up
	bool load_extensions = true;
	//! The maximum memory used by the database system (in bytes). Default: 80% of System available memory
	idx_t maximum_memory = (idx_t)-1;
	//! The maximum amount of CPU threads used by the database system. Default: all available.
	idx_t maximum_threads = (idx_t)-1;
	//! The number of external threads that work on DuckDB tasks. Default: none.
	idx_t external_threads = 0;
	//! Whether or not to create and use a temporary directory to store intermediates that do not fit in memory
	bool use_temporary_directory = true;
	//! Directory to store temporary structures that do not fit in memory
	string temporary_directory;
	//! The collation type of the database
	string collation = string();
	//! The order type used when none is specified (default: ASC)
	OrderType default_order_type = OrderType::ASCENDING;
	//! Null ordering used when none is specified (default: NULLS LAST)
	DefaultOrderByNullType default_null_order = DefaultOrderByNullType::NULLS_LAST;
	//! enable COPY and related commands
	bool enable_external_access = true;
	//! Whether or not object cache is used
	bool object_cache_enable = false;
	//! Whether or not the global http metadata cache is used
	bool http_metadata_cache_enable = false;
	//! Force checkpoint when CHECKPOINT is called or on shutdown, even if no changes have been made
	bool force_checkpoint = false;
	//! Run a checkpoint on successful shutdown and delete the WAL, to leave only a single database file behind
	bool checkpoint_on_shutdown = true;
	//! Debug flag that decides when a checkpoing should be aborted. Only used for testing purposes.
	CheckpointAbort checkpoint_abort = CheckpointAbort::NO_ABORT;
	//! Initialize the database with the standard set of DuckDB functions
	//! You should probably not touch this unless you know what you are doing
	bool initialize_default_database = true;
	//! The set of disabled optimizers (default empty)
	set<OptimizerType> disabled_optimizers;
	//! Force a specific compression method to be used when checkpointing (if available)
	CompressionType force_compression = CompressionType::COMPRESSION_AUTO;
	//! Force a specific bitpacking mode to be used when using the bitpacking compression method
	BitpackingMode force_bitpacking_mode = BitpackingMode::AUTO;
	//! Debug setting for window aggregation mode: (window, combine, separate)
	WindowAggregationMode window_mode = WindowAggregationMode::WINDOW;
	//! Whether or not preserving insertion order should be preserved
	bool preserve_insertion_order = true;
	//! Whether Arrow Arrays use Large or Regular buffers
	ArrowOffsetSize arrow_offset_size = ArrowOffsetSize::REGULAR;
	//! Database configuration variables as controlled by SET
	case_insensitive_map_t<Value> set_variables;
	//! Database configuration variable default values;
	case_insensitive_map_t<Value> set_variable_defaults;
	//! Directory to store extension binaries in
	string extension_directory;
	//! Whether unsigned extensions should be loaded
	bool allow_unsigned_extensions = false;
	//! Enable emitting FSST Vectors
	bool enable_fsst_vectors = false;
	//! Start transactions immediately in all attached databases - instead of lazily when a database is referenced
	bool immediate_transaction_mode = false;
	//! Debug setting - how to initialize  blocks in the storage layer when allocating
	DebugInitialize debug_initialize = DebugInitialize::NO_INITIALIZE;
	//! The set of unrecognized (other) options
	unordered_map<string, Value> unrecognized_options;
	//! Whether or not the configuration settings can be altered
	bool lock_configuration = false;
	//! Whether to print bindings when printing the plan (debug mode only)
	static bool debug_print_bindings;

	bool operator==(const DBConfigOptions &other) const;
};

struct DBConfig {
	friend class DatabaseInstance;
	friend class StorageManager;

public:
	DUCKDB_API DBConfig();
	DUCKDB_API DBConfig(std::unordered_map<string, string> &config_dict, bool read_only);
	DUCKDB_API ~DBConfig();

	mutex config_lock;
	//! Replacement table scans are automatically attempted when a table name cannot be found in the schema
	vector<ReplacementScan> replacement_scans;

	//! Extra parameters that can be SET for loaded extensions
	case_insensitive_map_t<ExtensionOption> extension_parameters;
	//! The FileSystem to use, can be overwritten to allow for injecting custom file systems for testing purposes (e.g.
	//! RamFS or something similar)
	unique_ptr<FileSystem> file_system;
	//! The allocator used by the system
	unique_ptr<Allocator> allocator;
	//! Database configuration options
	DBConfigOptions options;
	//! Extensions made to the parser
	vector<ParserExtension> parser_extensions;
	//! Extensions made to the optimizer
	vector<OptimizerExtension> optimizer_extensions;
	//! Error manager
	unique_ptr<ErrorManager> error_manager;
	//! A reference to the (shared) default allocator (Allocator::DefaultAllocator)
	shared_ptr<Allocator> default_allocator;
	//! Extensions made to binder
	vector<unique_ptr<OperatorExtension>> operator_extensions;
	//! Extensions made to storage
	case_insensitive_map_t<duckdb::unique_ptr<StorageExtension>> storage_extensions;
	//! A buffer pool can be shared across multiple databases (if desired).
	shared_ptr<BufferPool> buffer_pool;

public:
	DUCKDB_API static DBConfig &GetConfig(ClientContext &context);
	DUCKDB_API static DBConfig &GetConfig(DatabaseInstance &db);
	DUCKDB_API static DBConfig &Get(AttachedDatabase &db);
	DUCKDB_API static const DBConfig &GetConfig(const ClientContext &context);
	DUCKDB_API static const DBConfig &GetConfig(const DatabaseInstance &db);
	DUCKDB_API static vector<ConfigurationOption> GetOptions();
	DUCKDB_API static idx_t GetOptionCount();
	DUCKDB_API static vector<string> GetOptionNames();

	DUCKDB_API void AddExtensionOption(const string &name, string description, LogicalType parameter,
	                                   const Value &default_value = Value(), set_option_callback_t function = nullptr);
	//! Fetch an option by index. Returns a pointer to the option, or nullptr if out of range
	DUCKDB_API static ConfigurationOption *GetOptionByIndex(idx_t index);
	//! Fetch an option by name. Returns a pointer to the option, or nullptr if none exists.
	DUCKDB_API static ConfigurationOption *GetOptionByName(const string &name);

	DUCKDB_API void SetOption(const ConfigurationOption &option, const Value &value);
	DUCKDB_API void SetOption(DatabaseInstance *db, const ConfigurationOption &option, const Value &value);
	DUCKDB_API void SetOptionByName(const string &name, const Value &value);
	DUCKDB_API void ResetOption(DatabaseInstance *db, const ConfigurationOption &option);
	DUCKDB_API void SetOption(const string &name, Value value);
	DUCKDB_API void ResetOption(const string &name);

	DUCKDB_API static idx_t ParseMemoryLimit(const string &arg);

	//! Return the list of possible compression functions for the specific physical type
	DUCKDB_API vector<reference<CompressionFunction>> GetCompressionFunctions(PhysicalType data_type);
	//! Return the compression function for the specified compression type/physical type combo
	DUCKDB_API optional_ptr<CompressionFunction> GetCompressionFunction(CompressionType type, PhysicalType data_type);

	bool operator==(const DBConfig &other);
	bool operator!=(const DBConfig &other);

	DUCKDB_API CastFunctionSet &GetCastFunctions();
	void SetDefaultMaxThreads();
	void SetDefaultMaxMemory();

	OrderType ResolveOrder(OrderType order_type) const;
	OrderByNullType ResolveNullOrder(OrderType order_type, OrderByNullType null_type) const;

private:
	unique_ptr<CompressionFunctionSet> compression_functions;
	unique_ptr<CastFunctionSet> cast_functions;
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/main/valid_checker.hpp
//
//
//===----------------------------------------------------------------------===//







namespace duckdb {
class DatabaseInstance;
class MetaTransaction;

class ValidChecker {
public:
	ValidChecker();

	DUCKDB_API static ValidChecker &Get(DatabaseInstance &db);
	DUCKDB_API static ValidChecker &Get(MetaTransaction &transaction);

	DUCKDB_API void Invalidate(string error);
	DUCKDB_API bool IsInvalidated();
	DUCKDB_API string InvalidatedMessage();

	template <class T>
	static bool IsInvalidated(T &o) {
		return Get(o).IsInvalidated();
	}
	template <class T>
	static void Invalidate(T &o, string error) {
		Get(o).Invalidate(std::move(error));
	}

	template <class T>
	static string InvalidatedMessage(T &o) {
		return Get(o).InvalidatedMessage();
	}

private:
	//! Set to true if a fatal exception has occurred
	mutex invalidate_lock;
	atomic<bool> is_invalidated;
	string invalidated_msg;
};

} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/main/extension.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {
class DuckDB;

//! The Extension class is the base class used to define extensions
class Extension {
public:
	DUCKDB_API virtual ~Extension();

	DUCKDB_API virtual void Load(DuckDB &db) = 0;
	DUCKDB_API virtual std::string Name() = 0;
};

} // namespace duckdb


namespace duckdb {
class BufferManager;
class DatabaseManager;
class StorageManager;
class Catalog;
class TransactionManager;
class ConnectionManager;
class FileSystem;
class TaskScheduler;
class ObjectCache;
struct AttachInfo;

class DatabaseInstance : public std::enable_shared_from_this<DatabaseInstance> {
	friend class DuckDB;

public:
	DUCKDB_API DatabaseInstance();
	DUCKDB_API ~DatabaseInstance();

	DBConfig config;

public:
	BufferPool &GetBufferPool();
	DUCKDB_API BufferManager &GetBufferManager();
	DUCKDB_API DatabaseManager &GetDatabaseManager();
	DUCKDB_API FileSystem &GetFileSystem();
	DUCKDB_API TaskScheduler &GetScheduler();
	DUCKDB_API ObjectCache &GetObjectCache();
	DUCKDB_API ConnectionManager &GetConnectionManager();
	DUCKDB_API ValidChecker &GetValidChecker();
	DUCKDB_API void SetExtensionLoaded(const std::string &extension_name);

	idx_t NumberOfThreads();

	DUCKDB_API static DatabaseInstance &GetDatabase(ClientContext &context);

	DUCKDB_API const unordered_set<std::string> &LoadedExtensions();
	DUCKDB_API bool ExtensionIsLoaded(const std::string &name);

	DUCKDB_API bool TryGetCurrentSetting(const std::string &key, Value &result);

	unique_ptr<AttachedDatabase> CreateAttachedDatabase(AttachInfo &info, const string &type, AccessMode access_mode);

private:
	void Initialize(const char *path, DBConfig *config);
	void CreateMainDatabase();

	void Configure(DBConfig &config);

private:
	unique_ptr<BufferManager> buffer_manager;
	unique_ptr<DatabaseManager> db_manager;
	unique_ptr<TaskScheduler> scheduler;
	unique_ptr<ObjectCache> object_cache;
	unique_ptr<ConnectionManager> connection_manager;
	unordered_set<std::string> loaded_extensions;
	ValidChecker db_validity;
};

//! The database object. This object holds the catalog and all the
//! database-specific meta information.
class DuckDB {
public:
	DUCKDB_API explicit DuckDB(const char *path = nullptr, DBConfig *config = nullptr);
	DUCKDB_API explicit DuckDB(const string &path, DBConfig *config = nullptr);
	DUCKDB_API explicit DuckDB(DatabaseInstance &instance);

	DUCKDB_API ~DuckDB();

	//! Reference to the actual database instance
	shared_ptr<DatabaseInstance> instance;

public:
	template <class T>
	void LoadExtension() {
		T extension;
		if (ExtensionIsLoaded(extension.Name())) {
			return;
		}
		extension.Load(*this);
		instance->SetExtensionLoaded(extension.Name());
	}

	DUCKDB_API FileSystem &GetFileSystem();

	DUCKDB_API idx_t NumberOfThreads();
	DUCKDB_API static const char *SourceID();
	DUCKDB_API static const char *LibraryVersion();
	DUCKDB_API static idx_t StandardVectorSize();
	DUCKDB_API static string Platform();
	DUCKDB_API bool ExtensionIsLoaded(const std::string &name);
};

} // namespace duckdb


//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/main/appender.hpp
//
//
//===----------------------------------------------------------------------===//







namespace duckdb {

class ColumnDataCollection;
class ClientContext;
class DuckDB;
class TableCatalogEntry;
class Connection;

enum class AppenderType : uint8_t {
	LOGICAL, // Cast input -> LogicalType
	PHYSICAL // Cast input -> PhysicalType
};

//! The Appender class can be used to append elements to a table.
class BaseAppender {
protected:
	//! The amount of tuples that will be gathered in the column data collection before flushing
	static constexpr const idx_t FLUSH_COUNT = STANDARD_VECTOR_SIZE * 100;

	Allocator &allocator;
	//! The append types
	vector<LogicalType> types;
	//! The buffered data for the append
	unique_ptr<ColumnDataCollection> collection;
	//! Internal chunk used for appends
	DataChunk chunk;
	//! The current column to append to
	idx_t column = 0;
	//! The type of the appender
	AppenderType appender_type;

protected:
	DUCKDB_API BaseAppender(Allocator &allocator, AppenderType type);
	DUCKDB_API BaseAppender(Allocator &allocator, vector<LogicalType> types, AppenderType type);

public:
	DUCKDB_API virtual ~BaseAppender();

	//! Begins a new row append, after calling this the other AppendX() functions
	//! should be called the correct amount of times. After that,
	//! EndRow() should be called.
	DUCKDB_API void BeginRow();
	//! Finishes appending the current row.
	DUCKDB_API void EndRow();

	// Append functions
	template <class T>
	void Append(T value) {
		throw Exception("Undefined type for Appender::Append!");
	}

	DUCKDB_API void Append(const char *value, uint32_t length);

	// prepared statements
	template <typename... Args>
	void AppendRow(Args... args) {
		BeginRow();
		AppendRowRecursive(args...);
	}

	//! Commit the changes made by the appender.
	DUCKDB_API void Flush();
	//! Flush the changes made by the appender and close it. The appender cannot be used after this point
	DUCKDB_API void Close();

	vector<LogicalType> &GetTypes() {
		return types;
	}
	idx_t CurrentColumn() {
		return column;
	}
	DUCKDB_API void AppendDataChunk(DataChunk &value);

protected:
	void Destructor();
	virtual void FlushInternal(ColumnDataCollection &collection) = 0;
	void InitializeChunk();
	void FlushChunk();

	template <class T>
	void AppendValueInternal(T value);
	template <class SRC, class DST>
	void AppendValueInternal(Vector &vector, SRC input);
	template <class SRC, class DST>
	void AppendDecimalValueInternal(Vector &vector, SRC input);

	void AppendRowRecursive() {
		EndRow();
	}

	template <typename T, typename... Args>
	void AppendRowRecursive(T value, Args... args) {
		Append<T>(value);
		AppendRowRecursive(args...);
	}

	void AppendValue(const Value &value);
};

class Appender : public BaseAppender {
	//! A reference to a database connection that created this appender
	shared_ptr<ClientContext> context;
	//! The table description (including column names)
	unique_ptr<TableDescription> description;

public:
	DUCKDB_API Appender(Connection &con, const string &schema_name, const string &table_name);
	DUCKDB_API Appender(Connection &con, const string &table_name);
	DUCKDB_API ~Appender() override;

protected:
	void FlushInternal(ColumnDataCollection &collection) override;
};

class InternalAppender : public BaseAppender {
	//! The client context
	ClientContext &context;
	//! The internal table entry to append to
	TableCatalogEntry &table;

public:
	DUCKDB_API InternalAppender(ClientContext &context, TableCatalogEntry &table);
	DUCKDB_API ~InternalAppender() override;

protected:
	void FlushInternal(ColumnDataCollection &collection) override;
};

template <>
DUCKDB_API void BaseAppender::Append(bool value);
template <>
DUCKDB_API void BaseAppender::Append(int8_t value);
template <>
DUCKDB_API void BaseAppender::Append(int16_t value);
template <>
DUCKDB_API void BaseAppender::Append(int32_t value);
template <>
DUCKDB_API void BaseAppender::Append(int64_t value);
template <>
DUCKDB_API void BaseAppender::Append(hugeint_t value);
template <>
DUCKDB_API void BaseAppender::Append(uint8_t value);
template <>
DUCKDB_API void BaseAppender::Append(uint16_t value);
template <>
DUCKDB_API void BaseAppender::Append(uint32_t value);
template <>
DUCKDB_API void BaseAppender::Append(uint64_t value);
template <>
DUCKDB_API void BaseAppender::Append(float value);
template <>
DUCKDB_API void BaseAppender::Append(double value);
template <>
DUCKDB_API void BaseAppender::Append(date_t value);
template <>
DUCKDB_API void BaseAppender::Append(dtime_t value);
template <>
DUCKDB_API void BaseAppender::Append(timestamp_t value);
template <>
DUCKDB_API void BaseAppender::Append(interval_t value);
template <>
DUCKDB_API void BaseAppender::Append(const char *value);
template <>
DUCKDB_API void BaseAppender::Append(string_t value);
template <>
DUCKDB_API void BaseAppender::Append(Value value);
template <>
DUCKDB_API void BaseAppender::Append(std::nullptr_t value);

} // namespace duckdb

// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

/// \file adbc.h ADBC: Arrow Database connectivity
///
/// An Arrow-based interface between applications and database
/// drivers.  ADBC aims to provide a vendor-independent API for SQL
/// and Substrait-based database access that is targeted at
/// analytics/OLAP use cases.
///
/// This API is intended to be implemented directly by drivers and
/// used directly by client applications.  To assist portability
/// between different vendors, a "driver manager" library is also
/// provided, which implements this same API, but dynamically loads
/// drivers internally and forwards calls appropriately.
///
/// ADBC uses structs with free functions that operate on those
/// structs to model objects.
///
/// In general, objects allow serialized access from multiple threads,
/// but not concurrent access.  Specific implementations may permit
/// multiple threads.
///
/// \version 1.0.0



#include <stddef.h>
#include <stdint.h>


/// \defgroup Arrow C Data Interface
/// Definitions for the C Data Interface/C Stream Interface.
///
/// See https://arrow.apache.org/docs/format/CDataInterface.html
///
/// @{

//! @cond Doxygen_Suppress
namespace duckdb_adbc {
#ifdef __cplusplus
extern "C" {
#endif

#ifndef ARROW_ADBC
#define ARROW_ADBC

#ifndef ADBC_EXPORTING
#define ADBC_EXPORTING
#endif

// Storage class macros for Windows
// Allow overriding/aliasing with application-defined macros
#if !defined(ADBC_EXPORT)
#if defined(_WIN32)
#if defined(ADBC_EXPORTING)
#define ADBC_EXPORT __declspec(dllexport)
#else
#define ADBC_EXPORT __declspec(dllimport)
#endif // defined(ADBC_EXPORTING)
#else
#define ADBC_EXPORT
#endif // defined(_WIN32)
#endif // !defined(ADBC_EXPORT)

/// \defgroup adbc-error-handling Error Handling
/// ADBC uses integer error codes to signal errors. To provide more
/// detail about errors, functions may also return an AdbcError via an
/// optional out parameter, which can be inspected. If provided, it is
/// the responsibility of the caller to zero-initialize the AdbcError
/// value.
///
/// @{

/// \brief Error codes for operations that may fail.
typedef uint8_t AdbcStatusCode;

/// \brief No error.
#define ADBC_STATUS_OK 0
/// \brief An unknown error occurred.
///
/// May indicate a driver-side or database-side error.
#define ADBC_STATUS_UNKNOWN 1
/// \brief The operation is not implemented or supported.
///
/// May indicate a driver-side or database-side error.
#define ADBC_STATUS_NOT_IMPLEMENTED 2
/// \brief A requested resource was not found.
///
/// May indicate a driver-side or database-side error.
#define ADBC_STATUS_NOT_FOUND 3
/// \brief A requested resource already exists.
///
/// May indicate a driver-side or database-side error.
#define ADBC_STATUS_ALREADY_EXISTS 4
/// \brief The arguments are invalid, likely a programming error.
///
/// For instance, they may be of the wrong format, or out of range.
///
/// May indicate a driver-side or database-side error.
#define ADBC_STATUS_INVALID_ARGUMENT 5
/// \brief The preconditions for the operation are not met, likely a
///   programming error.
///
/// For instance, the object may be uninitialized, or may have not
/// been fully configured.
///
/// May indicate a driver-side or database-side error.
#define ADBC_STATUS_INVALID_STATE 6
/// \brief Invalid data was processed (not a programming error).
///
/// For instance, a division by zero may have occurred during query
/// execution.
///
/// May indicate a database-side error only.
#define ADBC_STATUS_INVALID_DATA 7
/// \brief The database's integrity was affected.
///
/// For instance, a foreign key check may have failed, or a uniqueness
/// constraint may have been violated.
///
/// May indicate a database-side error only.
#define ADBC_STATUS_INTEGRITY 8
/// \brief An error internal to the driver or database occurred.
///
/// May indicate a driver-side or database-side error.
#define ADBC_STATUS_INTERNAL 9
/// \brief An I/O error occurred.
///
/// For instance, a remote service may be unavailable.
///
/// May indicate a driver-side or database-side error.
#define ADBC_STATUS_IO 10
/// \brief The operation was cancelled, not due to a timeout.
///
/// May indicate a driver-side or database-side error.
#define ADBC_STATUS_CANCELLED 11
/// \brief The operation was cancelled due to a timeout.
///
/// May indicate a driver-side or database-side error.
#define ADBC_STATUS_TIMEOUT 12
/// \brief Authentication failed.
///
/// May indicate a database-side error only.
#define ADBC_STATUS_UNAUTHENTICATED 13
/// \brief The client is not authorized to perform the given operation.
///
/// May indicate a database-side error only.
#define ADBC_STATUS_UNAUTHORIZED 14

/// \brief A detailed error message for an operation.
struct ADBC_EXPORT AdbcError {
	/// \brief The error message.
	char *message;

	/// \brief A vendor-specific error code, if applicable.
	int32_t vendor_code;

	/// \brief A SQLSTATE error code, if provided, as defined by the
	///   SQL:2003 standard.  If not set, it should be set to
	///   "\0\0\0\0\0".
	char sqlstate[5];

	/// \brief Release the contained error.
	///
	/// Unlike other structures, this is an embedded callback to make it
	/// easier for the driver manager and driver to cooperate.
	void (*release)(struct AdbcError *error);
};

/// @}

/// \defgroup adbc-constants Constants
/// @{

/// \brief ADBC revision 1.0.0.
///
/// When passed to an AdbcDriverInitFunc(), the driver parameter must
/// point to an AdbcDriver.
#define ADBC_VERSION_1_0_0 1000000

/// \brief Canonical option value for enabling an option.
///
/// For use as the value in SetOption calls.
#define ADBC_OPTION_VALUE_ENABLED "true"
/// \brief Canonical option value for disabling an option.
///
/// For use as the value in SetOption calls.
#define ADBC_OPTION_VALUE_DISABLED "false"

/// \brief The database vendor/product name (e.g. the server name).
///   (type: utf8).
///
/// \see AdbcConnectionGetInfo
#define ADBC_INFO_VENDOR_NAME 0
/// \brief The database vendor/product version (type: utf8).
///
/// \see AdbcConnectionGetInfo
#define ADBC_INFO_VENDOR_VERSION 1
/// \brief The database vendor/product Arrow library version (type:
///   utf8).
///
/// \see AdbcConnectionGetInfo
#define ADBC_INFO_VENDOR_ARROW_VERSION 2

/// \brief The driver name (type: utf8).
///
/// \see AdbcConnectionGetInfo
#define ADBC_INFO_DRIVER_NAME 100
/// \brief The driver version (type: utf8).
///
/// \see AdbcConnectionGetInfo
#define ADBC_INFO_DRIVER_VERSION 101
/// \brief The driver Arrow library version (type: utf8).
///
/// \see AdbcConnectionGetInfo
#define ADBC_INFO_DRIVER_ARROW_VERSION 102

/// \brief Return metadata on catalogs, schemas, tables, and columns.
///
/// \see AdbcConnectionGetObjects
#define ADBC_OBJECT_DEPTH_ALL 0
/// \brief Return metadata on catalogs only.
///
/// \see AdbcConnectionGetObjects
#define ADBC_OBJECT_DEPTH_CATALOGS 1
/// \brief Return metadata on catalogs and schemas.
///
/// \see AdbcConnectionGetObjects
#define ADBC_OBJECT_DEPTH_DB_SCHEMAS 2
/// \brief Return metadata on catalogs, schemas, and tables.
///
/// \see AdbcConnectionGetObjects
#define ADBC_OBJECT_DEPTH_TABLES 3
/// \brief Return metadata on catalogs, schemas, tables, and columns.
///
/// \see AdbcConnectionGetObjects
#define ADBC_OBJECT_DEPTH_COLUMNS ADBC_OBJECT_DEPTH_ALL

/// \brief The name of the canonical option for whether autocommit is
///   enabled.
///
/// \see AdbcConnectionSetOption
#define ADBC_CONNECTION_OPTION_AUTOCOMMIT "adbc.connection.autocommit"

/// \brief The name of the canonical option for whether the current
///   connection should be restricted to being read-only.
///
/// \see AdbcConnectionSetOption
#define ADBC_CONNECTION_OPTION_READ_ONLY "adbc.connection.readonly"

/// \brief The name of the canonical option for setting the isolation
///   level of a transaction.
///
/// Should only be used in conjunction with autocommit disabled and
/// AdbcConnectionCommit / AdbcConnectionRollback. If the desired
/// isolation level is not supported by a driver, it should return an
/// appropriate error.
///
/// \see AdbcConnectionSetOption
#define ADBC_CONNECTION_OPTION_ISOLATION_LEVEL "adbc.connection.transaction.isolation_level"

/// \brief Use database or driver default isolation level
///
/// \see AdbcConnectionSetOption
#define ADBC_OPTION_ISOLATION_LEVEL_DEFAULT "adbc.connection.transaction.isolation.default"

/// \brief The lowest isolation level. Dirty reads are allowed, so one
///   transaction may see not-yet-committed changes made by others.
///
/// \see AdbcConnectionSetOption
#define ADBC_OPTION_ISOLATION_LEVEL_READ_UNCOMMITTED "adbc.connection.transaction.isolation.read_uncommitted"

/// \brief Lock-based concurrency control keeps write locks until the
///   end of the transaction, but read locks are released as soon as a
///   SELECT is performed. Non-repeatable reads can occur in this
///   isolation level.
///
/// More simply put, Read Committed is an isolation level that guarantees
/// that any data read is committed at the moment it is read. It simply
/// restricts the reader from seeing any intermediate, uncommitted,
/// 'dirty' reads. It makes no promise whatsoever that if the transaction
/// re-issues the read, it will find the same data; data is free to change
/// after it is read.
///
/// \see AdbcConnectionSetOption
#define ADBC_OPTION_ISOLATION_LEVEL_READ_COMMITTED "adbc.connection.transaction.isolation.read_committed"

/// \brief Lock-based concurrency control keeps read AND write locks
///   (acquired on selection data) until the end of the transaction.
///
/// However, range-locks are not managed, so phantom reads can occur.
/// Write skew is possible at this isolation level in some systems.
///
/// \see AdbcConnectionSetOption
#define ADBC_OPTION_ISOLATION_LEVEL_REPEATABLE_READ "adbc.connection.transaction.isolation.repeatable_read"

/// \brief This isolation guarantees that all reads in the transaction
///   will see a consistent snapshot of the database and the transaction
///   should only successfully commit if no updates conflict with any
///   concurrent updates made since that snapshot.
///
/// \see AdbcConnectionSetOption
#define ADBC_OPTION_ISOLATION_LEVEL_SNAPSHOT "adbc.connection.transaction.isolation.snapshot"

/// \brief Serializability requires read and write locks to be released
///   only at the end of the transaction. This includes acquiring range-
///   locks when a select query uses a ranged WHERE clause to avoid
///   phantom reads.
///
/// \see AdbcConnectionSetOption
#define ADBC_OPTION_ISOLATION_LEVEL_SERIALIZABLE "adbc.connection.transaction.isolation.serializable"

/// \brief The central distinction between serializability and linearizability
///   is that serializability is a global property; a property of an entire
///   history of operations and transactions. Linearizability is a local
///   property; a property of a single operation/transaction.
///
/// Linearizability can be viewed as a special case of strict serializability
/// where transactions are restricted to consist of a single operation applied
/// to a single object.
///
/// \see AdbcConnectionSetOption
#define ADBC_OPTION_ISOLATION_LEVEL_LINEARIZABLE "adbc.connection.transaction.isolation.linearizable"

/// \defgroup adbc-statement-ingestion Bulk Data Ingestion
/// While it is possible to insert data via prepared statements, it can
/// be more efficient to explicitly perform a bulk insert.  For
/// compatible drivers, this can be accomplished by setting up and
/// executing a statement.  Instead of setting a SQL query or Substrait
/// plan, bind the source data via AdbcStatementBind, and set the name
/// of the table to be created via AdbcStatementSetOption and the
/// options below.  Then, call AdbcStatementExecute with a NULL for
/// the out parameter (to indicate you do not expect a result set).
///
/// @{

/// \brief The name of the target table for a bulk insert.
///
/// The driver should attempt to create the table if it does not
/// exist.  If the table exists but has a different schema,
/// ADBC_STATUS_ALREADY_EXISTS should be raised.  Else, data should be
/// appended to the target table.
#define ADBC_INGEST_OPTION_TARGET_TABLE "adbc.ingest.target_table"
/// \brief Whether to create (the default) or append.
#define ADBC_INGEST_OPTION_MODE "adbc.ingest.mode"
/// \brief Create the table and insert data; error if the table exists.
#define ADBC_INGEST_OPTION_MODE_CREATE "adbc.ingest.mode.create"
/// \brief Do not create the table, and insert data; error if the
///   table does not exist (ADBC_STATUS_NOT_FOUND) or does not match
///   the schema of the data to append (ADBC_STATUS_ALREADY_EXISTS).
#define ADBC_INGEST_OPTION_MODE_APPEND "adbc.ingest.mode.append"

/// @}

/// @}

/// \defgroup adbc-database Database Initialization
/// Clients first initialize a database, then create a connection
/// (below).  This gives the implementation a place to initialize and
/// own any common connection state.  For example, in-memory databases
/// can place ownership of the actual database in this object.
/// @{

/// \brief An instance of a database.
///
/// Must be kept alive as long as any connections exist.
struct ADBC_EXPORT AdbcDatabase {
	/// \brief Opaque implementation-defined state.
	/// This field is NULLPTR iff the connection is unintialized/freed.
	void *private_data;
	/// \brief The associated driver (used by the driver manager to help
	///   track state).
	struct AdbcDriver *private_driver;
};

/// @}

/// \defgroup adbc-connection Connection Establishment
/// Functions for creating, using, and releasing database connections.
/// @{

/// \brief An active database connection.
///
/// Provides methods for query execution, managing prepared
/// statements, using transactions, and so on.
///
/// Connections are not required to be thread-safe, but they can be
/// used from multiple threads so long as clients take care to
/// serialize accesses to a connection.
struct ADBC_EXPORT AdbcConnection {
	/// \brief Opaque implementation-defined state.
	/// This field is NULLPTR iff the connection is unintialized/freed.
	void *private_data;
	/// \brief The associated driver (used by the driver manager to help
	///   track state).
	struct AdbcDriver *private_driver;
};

/// @}

/// \defgroup adbc-statement Managing Statements
/// Applications should first initialize a statement with
/// AdbcStatementNew. Then, the statement should be configured with
/// functions like AdbcStatementSetSqlQuery and
/// AdbcStatementSetOption. Finally, the statement can be executed
/// with AdbcStatementExecuteQuery (or call AdbcStatementPrepare first
/// to turn it into a prepared statement instead).
/// @{

/// \brief A container for all state needed to execute a database
/// query, such as the query itself, parameters for prepared
/// statements, driver parameters, etc.
///
/// Statements may represent queries or prepared statements.
///
/// Statements may be used multiple times and can be reconfigured
/// (e.g. they can be reused to execute multiple different queries).
/// However, executing a statement (and changing certain other state)
/// will invalidate result sets obtained prior to that execution.
///
/// Multiple statements may be created from a single connection.
/// However, the driver may block or error if they are used
/// concurrently (whether from a single thread or multiple threads).
///
/// Statements are not required to be thread-safe, but they can be
/// used from multiple threads so long as clients take care to
/// serialize accesses to a statement.
struct ADBC_EXPORT AdbcStatement {
	/// \brief Opaque implementation-defined state.
	/// This field is NULLPTR iff the connection is unintialized/freed.
	void *private_data;

	/// \brief The associated driver (used by the driver manager to help
	///   track state).
	struct AdbcDriver *private_driver;
};

/// \defgroup adbc-statement-partition Partitioned Results
/// Some backends may internally partition the results. These
/// partitions are exposed to clients who may wish to integrate them
/// with a threaded or distributed execution model, where partitions
/// can be divided among threads or machines and fetched in parallel.
///
/// To use partitioning, execute the statement with
/// AdbcStatementExecutePartitions to get the partition descriptors.
/// Call AdbcConnectionReadPartition to turn the individual
/// descriptors into ArrowArrayStream instances.  This may be done on
/// a different connection than the one the partition was created
/// with, or even in a different process on another machine.
///
/// Drivers are not required to support partitioning.
///
/// @{

/// \brief The partitions of a distributed/partitioned result set.
struct AdbcPartitions {
	/// \brief The number of partitions.
	size_t num_partitions;

	/// \brief The partitions of the result set, where each entry (up to
	///   num_partitions entries) is an opaque identifier that can be
	///   passed to AdbcConnectionReadPartition.
	const uint8_t **partitions;

	/// \brief The length of each corresponding entry in partitions.
	const size_t *partition_lengths;

	/// \brief Opaque implementation-defined state.
	/// This field is NULLPTR iff the connection is unintialized/freed.
	void *private_data;

	/// \brief Release the contained partitions.
	///
	/// Unlike other structures, this is an embedded callback to make it
	/// easier for the driver manager and driver to cooperate.
	void (*release)(struct AdbcPartitions *partitions);
};

/// @}

/// @}

/// \defgroup adbc-driver Driver Initialization
///
/// These functions are intended to help support integration between a
/// driver and the driver manager.
/// @{

/// \brief An instance of an initialized database driver.
///
/// This provides a common interface for vendor-specific driver
/// initialization routines. Drivers should populate this struct, and
/// applications can call ADBC functions through this struct, without
/// worrying about multiple definitions of the same symbol.
struct ADBC_EXPORT AdbcDriver {
	/// \brief Opaque driver-defined state.
	/// This field is NULL if the driver is unintialized/freed (but
	/// it need not have a value even if the driver is initialized).
	void *private_data;
	/// \brief Opaque driver manager-defined state.
	/// This field is NULL if the driver is unintialized/freed (but
	/// it need not have a value even if the driver is initialized).
	void *private_manager;

	/// \brief Release the driver and perform any cleanup.
	///
	/// This is an embedded callback to make it easier for the driver
	/// manager and driver to cooperate.
	AdbcStatusCode (*release)(struct AdbcDriver *driver, struct AdbcError *error);

	AdbcStatusCode (*DatabaseInit)(struct AdbcDatabase *, struct AdbcError *);
	AdbcStatusCode (*DatabaseNew)(struct AdbcDatabase *, struct AdbcError *);
	AdbcStatusCode (*DatabaseSetOption)(struct AdbcDatabase *, const char *, const char *, struct AdbcError *);
	AdbcStatusCode (*DatabaseRelease)(struct AdbcDatabase *, struct AdbcError *);

	AdbcStatusCode (*ConnectionCommit)(struct AdbcConnection *, struct AdbcError *);
	AdbcStatusCode (*ConnectionGetInfo)(struct AdbcConnection *, uint32_t *, size_t, struct ArrowArrayStream *,
	                                    struct AdbcError *);
	AdbcStatusCode (*ConnectionGetObjects)(struct AdbcConnection *, int, const char *, const char *, const char *,
	                                       const char **, const char *, struct ArrowArrayStream *, struct AdbcError *);
	AdbcStatusCode (*ConnectionGetTableSchema)(struct AdbcConnection *, const char *, const char *, const char *,
	                                           struct ArrowSchema *, struct AdbcError *);
	AdbcStatusCode (*ConnectionGetTableTypes)(struct AdbcConnection *, struct ArrowArrayStream *, struct AdbcError *);
	AdbcStatusCode (*ConnectionInit)(struct AdbcConnection *, struct AdbcDatabase *, struct AdbcError *);
	AdbcStatusCode (*ConnectionNew)(struct AdbcConnection *, struct AdbcError *);
	AdbcStatusCode (*ConnectionSetOption)(struct AdbcConnection *, const char *, const char *, struct AdbcError *);
	AdbcStatusCode (*ConnectionReadPartition)(struct AdbcConnection *, const uint8_t *, size_t,
	                                          struct ArrowArrayStream *, struct AdbcError *);
	AdbcStatusCode (*ConnectionRelease)(struct AdbcConnection *, struct AdbcError *);
	AdbcStatusCode (*ConnectionRollback)(struct AdbcConnection *, struct AdbcError *);

	AdbcStatusCode (*StatementBind)(struct AdbcStatement *, struct ArrowArray *, struct ArrowSchema *,
	                                struct AdbcError *);
	AdbcStatusCode (*StatementBindStream)(struct AdbcStatement *, struct ArrowArrayStream *, struct AdbcError *);
	AdbcStatusCode (*StatementExecuteQuery)(struct AdbcStatement *, struct ArrowArrayStream *, int64_t *,
	                                        struct AdbcError *);
	AdbcStatusCode (*StatementExecutePartitions)(struct AdbcStatement *, struct ArrowSchema *, struct AdbcPartitions *,
	                                             int64_t *, struct AdbcError *);
	AdbcStatusCode (*StatementGetParameterSchema)(struct AdbcStatement *, struct ArrowSchema *, struct AdbcError *);
	AdbcStatusCode (*StatementNew)(struct AdbcConnection *, struct AdbcStatement *, struct AdbcError *);
	AdbcStatusCode (*StatementPrepare)(struct AdbcStatement *, struct AdbcError *);
	AdbcStatusCode (*StatementRelease)(struct AdbcStatement *, struct AdbcError *);
	AdbcStatusCode (*StatementSetOption)(struct AdbcStatement *, const char *, const char *, struct AdbcError *);
	AdbcStatusCode (*StatementSetSqlQuery)(struct AdbcStatement *, const char *, struct AdbcError *);
	AdbcStatusCode (*StatementSetSubstraitPlan)(struct AdbcStatement *, const uint8_t *, size_t, struct AdbcError *);
};

/// @}

/// \addtogroup adbc-database
/// @{

/// \brief Allocate a new (but uninitialized) database.
///
/// Callers pass in a zero-initialized AdbcDatabase.
///
/// Drivers should allocate their internal data structure and set the private_data
/// field to point to the newly allocated struct. This struct should be released
/// when AdbcDatabaseRelease is called.
ADBC_EXPORT
AdbcStatusCode AdbcDatabaseNew(struct AdbcDatabase *database, struct AdbcError *error);

/// \brief Set a char* option.
///
/// Options may be set before AdbcDatabaseInit.  Some drivers may
/// support setting options after initialization as well.
///
/// \return ADBC_STATUS_NOT_IMPLEMENTED if the option is not recognized
ADBC_EXPORT
AdbcStatusCode AdbcDatabaseSetOption(struct AdbcDatabase *database, const char *key, const char *value,
                                     struct AdbcError *error);

/// \brief Finish setting options and initialize the database.
///
/// Some drivers may support setting options after initialization
/// as well.
ADBC_EXPORT
AdbcStatusCode AdbcDatabaseInit(struct AdbcDatabase *database, struct AdbcError *error);

/// \brief Destroy this database. No connections may exist.
/// \param[in] database The database to release.
/// \param[out] error An optional location to return an error
///   message if necessary.
ADBC_EXPORT
AdbcStatusCode AdbcDatabaseRelease(struct AdbcDatabase *database, struct AdbcError *error);

/// @}

/// \addtogroup adbc-connection
/// @{

/// \brief Allocate a new (but uninitialized) connection.
///
/// Callers pass in a zero-initialized AdbcConnection.
///
/// Drivers should allocate their internal data structure and set the private_data
/// field to point to the newly allocated struct. This struct should be released
/// when AdbcConnectionRelease is called.
ADBC_EXPORT
AdbcStatusCode AdbcConnectionNew(struct AdbcConnection *connection, struct AdbcError *error);

/// \brief Set a char* option.
///
/// Options may be set before AdbcConnectionInit.  Some drivers may
/// support setting options after initialization as well.
///
/// \return ADBC_STATUS_NOT_IMPLEMENTED if the option is not recognized
ADBC_EXPORT
AdbcStatusCode AdbcConnectionSetOption(struct AdbcConnection *connection, const char *key, const char *value,
                                       struct AdbcError *error);

/// \brief Finish setting options and initialize the connection.
///
/// Some drivers may support setting options after initialization
/// as well.
ADBC_EXPORT
AdbcStatusCode AdbcConnectionInit(struct AdbcConnection *connection, struct AdbcDatabase *database,
                                  struct AdbcError *error);

/// \brief Destroy this connection.
///
/// \param[in] connection The connection to release.
/// \param[out] error An optional location to return an error
///   message if necessary.
ADBC_EXPORT
AdbcStatusCode AdbcConnectionRelease(struct AdbcConnection *connection, struct AdbcError *error);

/// \defgroup adbc-connection-metadata Metadata
/// Functions for retrieving metadata about the database.
///
/// Generally, these functions return an ArrowArrayStream that can be
/// consumed to get the metadata as Arrow data.  The returned metadata
/// has an expected schema given in the function docstring. Schema
/// fields are nullable unless otherwise marked.  While no
/// AdbcStatement is used in these functions, the result set may count
/// as an active statement to the driver for the purposes of
/// concurrency management (e.g. if the driver has a limit on
/// concurrent active statements and it must execute a SQL query
/// internally in order to implement the metadata function).
///
/// Some functions accept "search pattern" arguments, which are
/// strings that can contain the special character "%" to match zero
/// or more characters, or "_" to match exactly one character.  (See
/// the documentation of DatabaseMetaData in JDBC or "Pattern Value
/// Arguments" in the ODBC documentation.)  Escaping is not currently
/// supported.
///
/// @{

/// \brief Get metadata about the database/driver.
///
/// The result is an Arrow dataset with the following schema:
///
/// Field Name                  | Field Type
/// ----------------------------|------------------------
/// info_name                   | uint32 not null
/// info_value                  | INFO_SCHEMA
///
/// INFO_SCHEMA is a dense union with members:
///
/// Field Name (Type Code)      | Field Type
/// ----------------------------|------------------------
/// string_value (0)            | utf8
/// bool_value (1)              | bool
/// int64_value (2)             | int64
/// int32_bitmask (3)           | int32
/// string_list (4)             | list<utf8>
/// int32_to_int32_list_map (5) | map<int32, list<int32>>
///
/// Each metadatum is identified by an integer code.  The recognized
/// codes are defined as constants.  Codes [0, 10_000) are reserved
/// for ADBC usage.  Drivers/vendors will ignore requests for
/// unrecognized codes (the row will be omitted from the result).
///
/// \param[in] connection The connection to query.
/// \param[in] info_codes A list of metadata codes to fetch, or NULL
///   to fetch all.
/// \param[in] info_codes_length The length of the info_codes
///   parameter.  Ignored if info_codes is NULL.
/// \param[out] out The result set.
/// \param[out] error Error details, if an error occurs.
ADBC_EXPORT
AdbcStatusCode AdbcConnectionGetInfo(struct AdbcConnection *connection, uint32_t *info_codes, size_t info_codes_length,
                                     struct ArrowArrayStream *out, struct AdbcError *error);

/// \brief Get a hierarchical view of all catalogs, database schemas,
///   tables, and columns.
///
/// The result is an Arrow dataset with the following schema:
///
/// | Field Name               | Field Type              |
/// |--------------------------|-------------------------|
/// | catalog_name             | utf8                    |
/// | catalog_db_schemas       | list<DB_SCHEMA_SCHEMA>  |
///
/// DB_SCHEMA_SCHEMA is a Struct with fields:
///
/// | Field Name               | Field Type              |
/// |--------------------------|-------------------------|
/// | db_schema_name           | utf8                    |
/// | db_schema_tables         | list<TABLE_SCHEMA>      |
///
/// TABLE_SCHEMA is a Struct with fields:
///
/// | Field Name               | Field Type              |
/// |--------------------------|-------------------------|
/// | table_name               | utf8 not null           |
/// | table_type               | utf8 not null           |
/// | table_columns            | list<COLUMN_SCHEMA>     |
/// | table_constraints        | list<CONSTRAINT_SCHEMA> |
///
/// COLUMN_SCHEMA is a Struct with fields:
///
/// | Field Name               | Field Type              | Comments |
/// |--------------------------|-------------------------|----------|
/// | column_name              | utf8 not null           |          |
/// | ordinal_position         | int32                   | (1)      |
/// | remarks                  | utf8                    | (2)      |
/// | xdbc_data_type           | int16                   | (3)      |
/// | xdbc_type_name           | utf8                    | (3)      |
/// | xdbc_column_size         | int32                   | (3)      |
/// | xdbc_decimal_digits      | int16                   | (3)      |
/// | xdbc_num_prec_radix      | int16                   | (3)      |
/// | xdbc_nullable            | int16                   | (3)      |
/// | xdbc_column_def          | utf8                    | (3)      |
/// | xdbc_sql_data_type       | int16                   | (3)      |
/// | xdbc_datetime_sub        | int16                   | (3)      |
/// | xdbc_char_octet_length   | int32                   | (3)      |
/// | xdbc_is_nullable         | utf8                    | (3)      |
/// | xdbc_scope_catalog       | utf8                    | (3)      |
/// | xdbc_scope_schema        | utf8                    | (3)      |
/// | xdbc_scope_table         | utf8                    | (3)      |
/// | xdbc_is_autoincrement    | bool                    | (3)      |
/// | xdbc_is_generatedcolumn  | bool                    | (3)      |
///
/// 1. The column's ordinal position in the table (starting from 1).
/// 2. Database-specific description of the column.
/// 3. Optional value.  Should be null if not supported by the driver.
///    xdbc_ values are meant to provide JDBC/ODBC-compatible metadata
///    in an agnostic manner.
///
/// CONSTRAINT_SCHEMA is a Struct with fields:
///
/// | Field Name               | Field Type              | Comments |
/// |--------------------------|-------------------------|----------|
/// | constraint_name          | utf8                    |          |
/// | constraint_type          | utf8 not null           | (1)      |
/// | constraint_column_names  | list<utf8> not null     | (2)      |
/// | constraint_column_usage  | list<USAGE_SCHEMA>      | (3)      |
///
/// 1. One of 'CHECK', 'FOREIGN KEY', 'PRIMARY KEY', or 'UNIQUE'.
/// 2. The columns on the current table that are constrained, in
///    order.
/// 3. For FOREIGN KEY only, the referenced table and columns.
///
/// USAGE_SCHEMA is a Struct with fields:
///
/// | Field Name               | Field Type              |
/// |--------------------------|-------------------------|
/// | fk_catalog               | utf8                    |
/// | fk_db_schema             | utf8                    |
/// | fk_table                 | utf8 not null           |
/// | fk_column_name           | utf8 not null           |
///
/// \param[in] connection The database connection.
/// \param[in] depth The level of nesting to display. If 0, display
///   all levels. If 1, display only catalogs (i.e.  catalog_schemas
///   will be null). If 2, display only catalogs and schemas
///   (i.e. db_schema_tables will be null), and so on.
/// \param[in] catalog Only show tables in the given catalog. If NULL,
///   do not filter by catalog. If an empty string, only show tables
///   without a catalog.  May be a search pattern (see section
///   documentation).
/// \param[in] db_schema Only show tables in the given database schema. If
///   NULL, do not filter by database schema. If an empty string, only show
///   tables without a database schema. May be a search pattern (see section
///   documentation).
/// \param[in] table_name Only show tables with the given name. If NULL, do not
///   filter by name. May be a search pattern (see section documentation).
/// \param[in] table_type Only show tables matching one of the given table
///   types. If NULL, show tables of any type. Valid table types can be fetched
///   from GetTableTypes.  Terminate the list with a NULL entry.
/// \param[in] column_name Only show columns with the given name. If
///   NULL, do not filter by name.  May be a search pattern (see
///   section documentation).
/// \param[out] out The result set.
/// \param[out] error Error details, if an error occurs.
ADBC_EXPORT
AdbcStatusCode AdbcConnectionGetObjects(struct AdbcConnection *connection, int depth, const char *catalog,
                                        const char *db_schema, const char *table_name, const char **table_type,
                                        const char *column_name, struct ArrowArrayStream *out, struct AdbcError *error);

/// \brief Get the Arrow schema of a table.
///
/// \param[in] connection The database connection.
/// \param[in] catalog The catalog (or nullptr if not applicable).
/// \param[in] db_schema The database schema (or nullptr if not applicable).
/// \param[in] table_name The table name.
/// \param[out] schema The table schema.
/// \param[out] error Error details, if an error occurs.
ADBC_EXPORT
AdbcStatusCode AdbcConnectionGetTableSchema(struct AdbcConnection *connection, const char *catalog,
                                            const char *db_schema, const char *table_name, struct ArrowSchema *schema,
                                            struct AdbcError *error);

/// \brief Get a list of table types in the database.
///
/// The result is an Arrow dataset with the following schema:
///
/// Field Name     | Field Type
/// ---------------|--------------
/// table_type     | utf8 not null
///
/// \param[in] connection The database connection.
/// \param[out] out The result set.
/// \param[out] error Error details, if an error occurs.
ADBC_EXPORT
AdbcStatusCode AdbcConnectionGetTableTypes(struct AdbcConnection *connection, struct ArrowArrayStream *out,
                                           struct AdbcError *error);

/// @}

/// \defgroup adbc-connection-partition Partitioned Results
/// Some databases may internally partition the results. These
/// partitions are exposed to clients who may wish to integrate them
/// with a threaded or distributed execution model, where partitions
/// can be divided among threads or machines for processing.
///
/// Drivers are not required to support partitioning.
///
/// Partitions are not ordered. If the result set is sorted,
/// implementations should return a single partition.
///
/// @{

/// \brief Construct a statement for a partition of a query. The
///   results can then be read independently.
///
/// A partition can be retrieved from AdbcPartitions.
///
/// \param[in] connection The connection to use.  This does not have
///   to be the same connection that the partition was created on.
/// \param[in] serialized_partition The partition descriptor.
/// \param[in] serialized_length The partition descriptor length.
/// \param[out] out The result set.
/// \param[out] error Error details, if an error occurs.
ADBC_EXPORT
AdbcStatusCode AdbcConnectionReadPartition(struct AdbcConnection *connection, const uint8_t *serialized_partition,
                                           size_t serialized_length, struct ArrowArrayStream *out,
                                           struct AdbcError *error);

/// @}

/// \defgroup adbc-connection-transaction Transaction Semantics
///
/// Connections start out in auto-commit mode by default (if
/// applicable for the given vendor). Use AdbcConnectionSetOption and
/// ADBC_CONNECTION_OPTION_AUTO_COMMIT to change this.
///
/// @{

/// \brief Commit any pending transactions. Only used if autocommit is
///   disabled.
///
/// Behavior is undefined if this is mixed with SQL transaction
/// statements.
ADBC_EXPORT
AdbcStatusCode AdbcConnectionCommit(struct AdbcConnection *connection, struct AdbcError *error);

/// \brief Roll back any pending transactions. Only used if autocommit
///   is disabled.
///
/// Behavior is undefined if this is mixed with SQL transaction
/// statements.
ADBC_EXPORT
AdbcStatusCode AdbcConnectionRollback(struct AdbcConnection *connection, struct AdbcError *error);

/// @}

/// @}

/// \addtogroup adbc-statement
/// @{

/// \brief Create a new statement for a given connection.
///
/// Callers pass in a zero-initialized AdbcStatement.
///
/// Drivers should allocate their internal data structure and set the private_data
/// field to point to the newly allocated struct. This struct should be released
/// when AdbcStatementRelease is called.
ADBC_EXPORT
AdbcStatusCode AdbcStatementNew(struct AdbcConnection *connection, struct AdbcStatement *statement,
                                struct AdbcError *error);

/// \brief Destroy a statement.
/// \param[in] statement The statement to release.
/// \param[out] error An optional location to return an error
///   message if necessary.
ADBC_EXPORT
AdbcStatusCode AdbcStatementRelease(struct AdbcStatement *statement, struct AdbcError *error);

/// \brief Execute a statement and get the results.
///
/// This invalidates any prior result sets.
///
/// \param[in] statement The statement to execute.
/// \param[out] out The results. Pass NULL if the client does not
///   expect a result set.
/// \param[out] rows_affected The number of rows affected if known,
///   else -1. Pass NULL if the client does not want this information.
/// \param[out] error An optional location to return an error
///   message if necessary.
ADBC_EXPORT
AdbcStatusCode AdbcStatementExecuteQuery(struct AdbcStatement *statement, struct ArrowArrayStream *out,
                                         int64_t *rows_affected, struct AdbcError *error);

/// \brief Turn this statement into a prepared statement to be
///   executed multiple times.
///
/// This invalidates any prior result sets.
ADBC_EXPORT
AdbcStatusCode AdbcStatementPrepare(struct AdbcStatement *statement, struct AdbcError *error);

/// \defgroup adbc-statement-sql SQL Semantics
/// Functions for executing SQL queries, or querying SQL-related
/// metadata. Drivers are not required to support both SQL and
/// Substrait semantics. If they do, it may be via converting
/// between representations internally.
/// @{

/// \brief Set the SQL query to execute.
///
/// The query can then be executed with AdbcStatementExecute.  For
/// queries expected to be executed repeatedly, AdbcStatementPrepare
/// the statement first.
///
/// \param[in] statement The statement.
/// \param[in] query The query to execute.
/// \param[out] error Error details, if an error occurs.
ADBC_EXPORT
AdbcStatusCode AdbcStatementSetSqlQuery(struct AdbcStatement *statement, const char *query, struct AdbcError *error);

/// @}

/// \defgroup adbc-statement-substrait Substrait Semantics
/// Functions for executing Substrait plans, or querying
/// Substrait-related metadata.  Drivers are not required to support
/// both SQL and Substrait semantics.  If they do, it may be via
/// converting between representations internally.
/// @{

/// \brief Set the Substrait plan to execute.
///
/// The query can then be executed with AdbcStatementExecute.  For
/// queries expected to be executed repeatedly, AdbcStatementPrepare
/// the statement first.
///
/// \param[in] statement The statement.
/// \param[in] plan The serialized substrait.Plan to execute.
/// \param[in] length The length of the serialized plan.
/// \param[out] error Error details, if an error occurs.
ADBC_EXPORT
AdbcStatusCode AdbcStatementSetSubstraitPlan(struct AdbcStatement *statement, const uint8_t *plan, size_t length,
                                             struct AdbcError *error);

/// @}

/// \brief Bind Arrow data. This can be used for bulk inserts or
///   prepared statements.
///
/// \param[in] statement The statement to bind to.
/// \param[in] values The values to bind. The driver will call the
///   release callback itself, although it may not do this until the
///   statement is released.
/// \param[in] schema The schema of the values to bind.
/// \param[out] error An optional location to return an error message
///   if necessary.
ADBC_EXPORT
AdbcStatusCode AdbcStatementBind(struct AdbcStatement *statement, struct ArrowArray *values, struct ArrowSchema *schema,
                                 struct AdbcError *error);

/// \brief Bind Arrow data. This can be used for bulk inserts or
///   prepared statements.
/// \param[in] statement The statement to bind to.
/// \param[in] stream The values to bind. The driver will call the
///   release callback itself, although it may not do this until the
///   statement is released.
/// \param[out] error An optional location to return an error message
///   if necessary.
ADBC_EXPORT
AdbcStatusCode AdbcStatementBindStream(struct AdbcStatement *statement, struct ArrowArrayStream *stream,
                                       struct AdbcError *error);

/// \brief Get the schema for bound parameters.
///
/// This retrieves an Arrow schema describing the number, names, and
/// types of the parameters in a parameterized statement.  The fields
/// of the schema should be in order of the ordinal position of the
/// parameters; named parameters should appear only once.
///
/// If the parameter does not have a name, or the name cannot be
/// determined, the name of the corresponding field in the schema will
/// be an empty string.  If the type cannot be determined, the type of
/// the corresponding field will be NA (NullType).
///
/// This should be called after AdbcStatementPrepare.
///
/// \return ADBC_STATUS_NOT_IMPLEMENTED if the schema cannot be determined.
ADBC_EXPORT
AdbcStatusCode AdbcStatementGetParameterSchema(struct AdbcStatement *statement, struct ArrowSchema *schema,
                                               struct AdbcError *error);

/// \brief Set a string option on a statement.
ADBC_EXPORT
AdbcStatusCode AdbcStatementSetOption(struct AdbcStatement *statement, const char *key, const char *value,
                                      struct AdbcError *error);

/// \addtogroup adbc-statement-partition
/// @{

/// \brief Execute a statement and get the results as a partitioned
///   result set.
///
/// \param[in] statement The statement to execute.
/// \param[out] schema The schema of the result set.
/// \param[out] partitions The result partitions.
/// \param[out] rows_affected The number of rows affected if known,
///   else -1. Pass NULL if the client does not want this information.
/// \param[out] error An optional location to return an error
///   message if necessary.
/// \return ADBC_STATUS_NOT_IMPLEMENTED if the driver does not support
///   partitioned results
ADBC_EXPORT
AdbcStatusCode AdbcStatementExecutePartitions(struct AdbcStatement *statement, struct ArrowSchema *schema,
                                              struct AdbcPartitions *partitions, int64_t *rows_affected,
                                              struct AdbcError *error);

/// @}

/// @}

/// \addtogroup adbc-driver
/// @{

/// \brief Common entry point for drivers via the driver manager
///   (which uses dlopen(3)/LoadLibrary). The driver manager is told
///   to load a library and call a function of this type to load the
///   driver.
///
/// Although drivers may choose any name for this function, the
/// recommended name is "AdbcDriverInit".
///
/// \param[in] version The ADBC revision to attempt to initialize (see
///   ADBC_VERSION_1_0_0).
/// \param[out] driver The table of function pointers to
///   initialize. Should be a pointer to the appropriate struct for
///   the given version (see the documentation for the version).
/// \param[out] error An optional location to return an error message
///   if necessary.
/// \return ADBC_STATUS_OK if the driver was initialized, or
///   ADBC_STATUS_NOT_IMPLEMENTED if the version is not supported.  In
///   that case, clients may retry with a different version.
typedef AdbcStatusCode (*AdbcDriverInitFunc)(int version, void *driver, struct AdbcError *error);

/// @}

#endif // ADBC

#ifdef __cplusplus
}
#endif
} // namespace duckdb_adbc
//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/adbc/adbc.hpp
//
//
//===----------------------------------------------------------------------===//





#include <string>

namespace duckdb_adbc {

AdbcStatusCode DatabaseNew(struct AdbcDatabase *database, struct AdbcError *error);

AdbcStatusCode DatabaseSetOption(struct AdbcDatabase *database, const char *key, const char *value,
                                 struct AdbcError *error);

AdbcStatusCode DatabaseInit(struct AdbcDatabase *database, struct AdbcError *error);

AdbcStatusCode DatabaseRelease(struct AdbcDatabase *database, struct AdbcError *error);

AdbcStatusCode ConnectionNew(struct AdbcConnection *connection, struct AdbcError *error);

AdbcStatusCode ConnectionSetOption(struct AdbcConnection *connection, const char *key, const char *value,
                                   struct AdbcError *error);

AdbcStatusCode ConnectionInit(struct AdbcConnection *connection, struct AdbcDatabase *database,
                              struct AdbcError *error);

AdbcStatusCode ConnectionRelease(struct AdbcConnection *connection, struct AdbcError *error);

AdbcStatusCode ConnectionGetInfo(struct AdbcConnection *connection, uint32_t *info_codes, size_t info_codes_length,
                                 struct ArrowArrayStream *out, struct AdbcError *error);

AdbcStatusCode ConnectionGetObjects(struct AdbcConnection *connection, int depth, const char *catalog,
                                    const char *db_schema, const char *table_name, const char **table_type,
                                    const char *column_name, struct ArrowArrayStream *out, struct AdbcError *error);

AdbcStatusCode ConnectionGetTableSchema(struct AdbcConnection *connection, const char *catalog, const char *db_schema,
                                        const char *table_name, struct ArrowSchema *schema, struct AdbcError *error);

AdbcStatusCode ConnectionGetTableTypes(struct AdbcConnection *connection, struct ArrowArrayStream *out,
                                       struct AdbcError *error);

AdbcStatusCode ConnectionReadPartition(struct AdbcConnection *connection, const uint8_t *serialized_partition,
                                       size_t serialized_length, struct ArrowArrayStream *out, struct AdbcError *error);

AdbcStatusCode ConnectionCommit(struct AdbcConnection *connection, struct AdbcError *error);

AdbcStatusCode ConnectionRollback(struct AdbcConnection *connection, struct AdbcError *error);

AdbcStatusCode StatementNew(struct AdbcConnection *connection, struct AdbcStatement *statement,
                            struct AdbcError *error);

AdbcStatusCode StatementRelease(struct AdbcStatement *statement, struct AdbcError *error);

AdbcStatusCode StatementExecuteQuery(struct AdbcStatement *statement, struct ArrowArrayStream *out,
                                     int64_t *rows_affected, struct AdbcError *error);

AdbcStatusCode StatementPrepare(struct AdbcStatement *statement, struct AdbcError *error);

AdbcStatusCode StatementSetSqlQuery(struct AdbcStatement *statement, const char *query, struct AdbcError *error);

AdbcStatusCode StatementSetSubstraitPlan(struct AdbcStatement *statement, const uint8_t *plan, size_t length,
                                         struct AdbcError *error);

AdbcStatusCode StatementBind(struct AdbcStatement *statement, struct ArrowArray *values, struct ArrowSchema *schema,
                             struct AdbcError *error);

AdbcStatusCode StatementBindStream(struct AdbcStatement *statement, struct ArrowArrayStream *stream,
                                   struct AdbcError *error);

AdbcStatusCode StatementGetParameterSchema(struct AdbcStatement *statement, struct ArrowSchema *schema,
                                           struct AdbcError *error);

AdbcStatusCode StatementSetOption(struct AdbcStatement *statement, const char *key, const char *value,
                                  struct AdbcError *error);

AdbcStatusCode StatementExecutePartitions(struct AdbcStatement *statement, struct ArrowSchema *schema,
                                          struct AdbcPartitions *partitions, int64_t *rows_affected,
                                          struct AdbcError *error);

void SetError(struct AdbcError *error, const std::string &message);

void InitiliazeADBCError(AdbcError *error);
} // namespace duckdb_adbc
//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/arrow/arrow_converter.hpp
//
//
//===----------------------------------------------------------------------===//







struct ArrowSchema;

namespace duckdb {

struct ArrowConverter {
	DUCKDB_API static void ToArrowSchema(ArrowSchema *out_schema, const vector<LogicalType> &types,
	                                     const vector<string> &names, const ArrowOptions &options);
	DUCKDB_API static void ToArrowArray(DataChunk &input, ArrowArray *out_array, ArrowOptions options);
};

} // namespace duckdb
//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/types/blob.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {

//! The Blob class is a static class that holds helper functions for the Blob type.
class Blob {
public:
	// map of integer -> hex value
	static constexpr const char *HEX_TABLE = "0123456789ABCDEF";
	// reverse map of byte -> integer value, or -1 for invalid hex values
	static const int HEX_MAP[256];
	//! map of index -> base64 character
	static constexpr const char *BASE64_MAP = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
	//! padding character used in base64 encoding
	static constexpr const char BASE64_PADDING = '=';

public:
	//! Returns the string size of a blob -> string conversion
	DUCKDB_API static idx_t GetStringSize(string_t blob);
	//! Converts a blob to a string, writing the output to the designated output string.
	//! The string needs to have space for at least GetStringSize(blob) bytes.
	DUCKDB_API static void ToString(string_t blob, char *output);
	//! Convert a blob object to a string
	DUCKDB_API static string ToString(string_t blob);

	//! Returns the blob size of a string -> blob conversion
	DUCKDB_API static bool TryGetBlobSize(string_t str, idx_t &result_size, string *error_message);
	DUCKDB_API static idx_t GetBlobSize(string_t str);
	//! Convert a string to a blob. This function should ONLY be called after calling GetBlobSize, since it does NOT
	//! perform data validation.
	DUCKDB_API static void ToBlob(string_t str, data_ptr_t output);
	//! Convert a string object to a blob
	DUCKDB_API static string ToBlob(string_t str);

	// base 64 conversion functions
	//! Returns the string size of a blob -> base64 conversion
	DUCKDB_API static idx_t ToBase64Size(string_t blob);
	//! Converts a blob to a base64 string, output should have space for at least ToBase64Size(blob) bytes
	DUCKDB_API static void ToBase64(string_t blob, char *output);

	//! Returns the string size of a base64 string -> blob conversion
	DUCKDB_API static idx_t FromBase64Size(string_t str);
	//! Converts a base64 string to a blob, output should have space for at least FromBase64Size(blob) bytes
	DUCKDB_API static void FromBase64(string_t str, data_ptr_t output, idx_t output_size);
};
} // namespace duckdb
//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/types/decimal.hpp
//
//
//===----------------------------------------------------------------------===//





namespace duckdb {

template <class PHYSICAL_TYPE>
struct DecimalWidth {};

template <>
struct DecimalWidth<int16_t> {
	static constexpr uint8_t max = 4;
};

template <>
struct DecimalWidth<int32_t> {
	static constexpr uint8_t max = 9;
};

template <>
struct DecimalWidth<int64_t> {
	static constexpr uint8_t max = 18;
};

template <>
struct DecimalWidth<hugeint_t> {
	static constexpr uint8_t max = 38;
};

//! The Decimal class is a static class that holds helper functions for the Decimal type
class Decimal {
public:
	static constexpr uint8_t MAX_WIDTH_INT16 = DecimalWidth<int16_t>::max;
	static constexpr uint8_t MAX_WIDTH_INT32 = DecimalWidth<int32_t>::max;
	static constexpr uint8_t MAX_WIDTH_INT64 = DecimalWidth<int64_t>::max;
	static constexpr uint8_t MAX_WIDTH_INT128 = DecimalWidth<hugeint_t>::max;
	static constexpr uint8_t MAX_WIDTH_DECIMAL = MAX_WIDTH_INT128;

public:
	static string ToString(int16_t value, uint8_t width, uint8_t scale);
	static string ToString(int32_t value, uint8_t width, uint8_t scale);
	static string ToString(int64_t value, uint8_t width, uint8_t scale);
	static string ToString(hugeint_t value, uint8_t width, uint8_t scale);
};
} // namespace duckdb
//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/types/uuid.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {
class ClientContext;
struct RandomEngine;

//! The UUID class contains static operations for the UUID type
class UUID {
public:
	constexpr static const uint8_t STRING_SIZE = 36;
	//! Convert a uuid string to a hugeint object
	static bool FromString(string str, hugeint_t &result);
	//! Convert a uuid string to a hugeint object
	static bool FromCString(const char *str, idx_t len, hugeint_t &result) {
		return FromString(string(str, 0, len), result);
	}
	//! Convert a hugeint object to a uuid style string
	static void ToString(hugeint_t input, char *buf);

	//! Convert a hugeint object to a uuid style string
	static hugeint_t GenerateRandomUUID(RandomEngine &engine);
	static hugeint_t GenerateRandomUUID();

	//! Convert a hugeint object to a uuid style string
	static string ToString(hugeint_t input) {
		char buff[STRING_SIZE];
		ToString(input, buff);
		return string(buff, STRING_SIZE);
	}

	static hugeint_t FromString(string str) {
		hugeint_t result;
		FromString(str, result);
		return result;
	}
};

} // namespace duckdb
//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/common/types/time.hpp
//
//
//===----------------------------------------------------------------------===//







namespace duckdb {

struct dtime_t;

//! The Time class is a static class that holds helper functions for the Time
//! type.
class Time {
public:
	//! Convert a string in the format "hh:mm:ss" to a time object
	DUCKDB_API static dtime_t FromString(const string &str, bool strict = false);
	DUCKDB_API static dtime_t FromCString(const char *buf, idx_t len, bool strict = false);
	DUCKDB_API static bool TryConvertTime(const char *buf, idx_t len, idx_t &pos, dtime_t &result, bool strict = false);

	//! Convert a time object to a string in the format "hh:mm:ss"
	DUCKDB_API static string ToString(dtime_t time);
	//! Convert a UTC offset to ±HH[:MM]
	DUCKDB_API static string ToUTCOffset(int hour_offset, int minute_offset);

	DUCKDB_API static dtime_t FromTime(int32_t hour, int32_t minute, int32_t second, int32_t microseconds = 0);

	//! Extract the time from a given timestamp object
	DUCKDB_API static void Convert(dtime_t time, int32_t &out_hour, int32_t &out_min, int32_t &out_sec,
	                               int32_t &out_micros);

	DUCKDB_API static string ConversionError(const string &str);
	DUCKDB_API static string ConversionError(string_t str);

	DUCKDB_API static dtime_t FromTimeMs(int64_t time_ms);
	DUCKDB_API static dtime_t FromTimeNs(int64_t time_ns);

	DUCKDB_API static bool IsValidTime(int32_t hour, int32_t minute, int32_t second, int32_t microseconds);

private:
	static bool TryConvertInternal(const char *buf, idx_t len, idx_t &pos, dtime_t &result, bool strict);
};

} // namespace duckdb
//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/parsed_data/create_table_function_info.hpp
//
//
//===----------------------------------------------------------------------===//



//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/parsed_data/create_function_info.hpp
//
//
//===----------------------------------------------------------------------===//






namespace duckdb {

struct CreateFunctionInfo : public CreateInfo {
	explicit CreateFunctionInfo(CatalogType type, string schema = DEFAULT_SCHEMA) : CreateInfo(type, schema) {
		D_ASSERT(type == CatalogType::SCALAR_FUNCTION_ENTRY || type == CatalogType::AGGREGATE_FUNCTION_ENTRY ||
		         type == CatalogType::TABLE_FUNCTION_ENTRY || type == CatalogType::PRAGMA_FUNCTION_ENTRY ||
		         type == CatalogType::MACRO_ENTRY || type == CatalogType::TABLE_MACRO_ENTRY);
	}

	//! Function name
	string name;
	//! The description (if any)
	string description;
	//! Parameter names (if any)
	vector<string> parameter_names;
	//! The example (if any)
	string example;

protected:
	void SerializeInternal(Serializer &serializer) const override {
		serializer.WriteString(name);
	}
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/function/function_set.hpp
//
//
//===----------------------------------------------------------------------===//






//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/function/pragma_function.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/parsed_data/pragma_info.hpp
//
//
//===----------------------------------------------------------------------===//








namespace duckdb {

enum class PragmaType : uint8_t { PRAGMA_STATEMENT, PRAGMA_CALL };

struct PragmaInfo : public ParseInfo {
	//! Name of the PRAGMA statement
	string name;
	//! Parameter list (if any)
	vector<Value> parameters;
	//! Named parameter list (if any)
	named_parameter_map_t named_parameters;

public:
	unique_ptr<PragmaInfo> Copy() const {
		auto result = make_uniq<PragmaInfo>();
		result->name = name;
		result->parameters = parameters;
		result->named_parameters = named_parameters;
		return result;
	}
};

} // namespace duckdb



namespace duckdb {
class ClientContext;

//! Return a substitute query to execute instead of this pragma statement
typedef string (*pragma_query_t)(ClientContext &context, const FunctionParameters &parameters);
//! Execute the main pragma function
typedef void (*pragma_function_t)(ClientContext &context, const FunctionParameters &parameters);

//! Pragma functions are invoked by calling PRAGMA x
//! Pragma functions come in three types:
//! * Call: function call, e.g. PRAGMA table_info('tbl')
//!   -> call statements can take multiple parameters
//! * Statement: statement without parameters, e.g. PRAGMA show_tables
//!   -> this is similar to a call pragma but without parameters
//! Pragma functions can either return a new query to execute (pragma_query_t)
//! or they can
class PragmaFunction : public SimpleNamedParameterFunction {
public:
	// Call
	DUCKDB_API static PragmaFunction PragmaCall(const string &name, pragma_query_t query, vector<LogicalType> arguments,
	                                            LogicalType varargs = LogicalType::INVALID);
	DUCKDB_API static PragmaFunction PragmaCall(const string &name, pragma_function_t function,
	                                            vector<LogicalType> arguments,
	                                            LogicalType varargs = LogicalType::INVALID);
	// Statement
	DUCKDB_API static PragmaFunction PragmaStatement(const string &name, pragma_query_t query);
	DUCKDB_API static PragmaFunction PragmaStatement(const string &name, pragma_function_t function);

	DUCKDB_API string ToString() const override;

public:
	PragmaType type;

	pragma_query_t query;
	pragma_function_t function;
	named_parameter_type_map_t named_parameters;

private:
	PragmaFunction(string name, PragmaType pragma_type, pragma_query_t query, pragma_function_t function,
	               vector<LogicalType> arguments, LogicalType varargs);
};

} // namespace duckdb


namespace duckdb {

template <class T>
class FunctionSet {
public:
	explicit FunctionSet(string name) : name(name) {
	}

	//! The name of the function set
	string name;
	//! The set of functions.
	vector<T> functions;

public:
	void AddFunction(T function) {
		functions.push_back(std::move(function));
	}
	idx_t Size() {
		return functions.size();
	}
	T GetFunctionByOffset(idx_t offset) {
		D_ASSERT(offset < functions.size());
		return functions[offset];
	}
	T &GetFunctionReferenceByOffset(idx_t offset) {
		D_ASSERT(offset < functions.size());
		return functions[offset];
	}
	bool MergeFunctionSet(FunctionSet<T> new_functions) {
		D_ASSERT(!new_functions.functions.empty());
		bool need_rewrite_entry = false;
		for (auto &new_func : new_functions.functions) {
			bool can_add = true;
			for (auto &func : functions) {
				if (new_func.Equal(func)) {
					can_add = false;
					break;
				}
			}
			if (can_add) {
				functions.push_back(new_func);
				need_rewrite_entry = true;
			}
		}
		return need_rewrite_entry;
	}
};

class ScalarFunctionSet : public FunctionSet<ScalarFunction> {
public:
	DUCKDB_API explicit ScalarFunctionSet();
	DUCKDB_API explicit ScalarFunctionSet(string name);
	DUCKDB_API explicit ScalarFunctionSet(ScalarFunction fun);

	DUCKDB_API ScalarFunction GetFunctionByArguments(ClientContext &context, const vector<LogicalType> &arguments);
};

class AggregateFunctionSet : public FunctionSet<AggregateFunction> {
public:
	DUCKDB_API explicit AggregateFunctionSet();
	DUCKDB_API explicit AggregateFunctionSet(string name);
	DUCKDB_API explicit AggregateFunctionSet(AggregateFunction fun);

	DUCKDB_API AggregateFunction GetFunctionByArguments(ClientContext &context, const vector<LogicalType> &arguments);
};

class TableFunctionSet : public FunctionSet<TableFunction> {
public:
	DUCKDB_API explicit TableFunctionSet(string name);
	DUCKDB_API explicit TableFunctionSet(TableFunction fun);

	TableFunction GetFunctionByArguments(ClientContext &context, const vector<LogicalType> &arguments);
};

class PragmaFunctionSet : public FunctionSet<PragmaFunction> {
public:
	DUCKDB_API explicit PragmaFunctionSet(string name);
	DUCKDB_API explicit PragmaFunctionSet(PragmaFunction fun);
};

} // namespace duckdb


namespace duckdb {

struct CreateTableFunctionInfo : public CreateFunctionInfo {
	DUCKDB_API explicit CreateTableFunctionInfo(TableFunction function);
	DUCKDB_API explicit CreateTableFunctionInfo(TableFunctionSet set);

	//! The table functions
	TableFunctionSet functions;

public:
	DUCKDB_API unique_ptr<CreateInfo> Copy() const override;
	DUCKDB_API unique_ptr<AlterInfo> GetAlterInfo() const override;
};

} // namespace duckdb
//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/parsed_data/create_copy_function_info.hpp
//
//
//===----------------------------------------------------------------------===//




//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/function/copy_function.hpp
//
//
//===----------------------------------------------------------------------===//





//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/parsed_data/copy_info.hpp
//
//
//===----------------------------------------------------------------------===//









namespace duckdb {

struct CopyInfo : public ParseInfo {
	CopyInfo() : catalog(INVALID_CATALOG), schema(DEFAULT_SCHEMA) {
	}

	//! The catalog name to copy to/from
	string catalog;
	//! The schema name to copy to/from
	string schema;
	//! The table name to copy to/from
	string table;
	//! List of columns to copy to/from
	vector<string> select_list;
	//! Whether or not this is a copy to file (false) or copy from a file (true)
	bool is_from;
	//! The file format of the external file
	string format;
	//! The file path to copy to/from
	string file_path;
	//! Set of (key, value) options
	case_insensitive_map_t<vector<Value>> options;

public:
	unique_ptr<CopyInfo> Copy() const {
		auto result = make_uniq<CopyInfo>();
		result->catalog = catalog;
		result->schema = schema;
		result->table = table;
		result->select_list = select_list;
		result->file_path = file_path;
		result->is_from = is_from;
		result->format = format;
		result->options = options;
		return result;
	}
};

} // namespace duckdb

//===----------------------------------------------------------------------===//
//                         DuckDB
//
// duckdb/parser/statement/copy_statement.hpp
//
//
//===----------------------------------------------------------------------===//







namespace duckdb {

class CopyStatement : public SQLStatement {
public:
	static constexpr const StatementType TYPE = StatementType::COPY_STATEMENT;

public:
	CopyStatement();

	unique_ptr<CopyInfo> info;
	// The SQL statement used instead of a table when copying data out to a file
	unique_ptr<QueryNode> select_statement;
	string ToString() const override;
	string CopyOptionsToString(const string &format, const case_insensitive_map_t<vector<Value>> &options) const;

protected:
	CopyStatement(const CopyStatement &other);

public:
	DUCKDB_API unique_ptr<SQLStatement> Copy() const override;

private:
};
} // namespace duckdb


namespace duckdb {

class Binder;
struct BoundStatement;
class ColumnDataCollection;
class ExecutionContext;

struct LocalFunctionData {
	virtual ~LocalFunctionData() {
	}

	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}
	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
};

struct GlobalFunctionData {
	virtual ~GlobalFunctionData() {
	}

	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}
	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
};

struct PreparedBatchData {
	virtual ~PreparedBatchData() {
	}

	template <class TARGET>
	TARGET &Cast() {
		D_ASSERT(dynamic_cast<TARGET *>(this));
		return reinterpret_cast<TARGET &>(*this);
	}
	template <class TARGET>
	const TARGET &Cast() const {
		D_ASSERT(dynamic_cast<const TARGET *>(this));
		return reinterpret_cast<const TARGET &>(*this);
	}
};

enum class CopyFunctionExecutionMode { REGULAR_COPY_TO_FILE, PARALLEL_COPY_TO_FILE, BATCH_COPY_TO_FILE };

typedef BoundStatement (*copy_to_plan_t)(Binder &binder, CopyStatement &stmt);
typedef unique_ptr<FunctionData> (*copy_to_bind_t)(ClientContext &context, CopyInfo &info, vector<string> &names,
                                                   vector<LogicalType> &sql_types);
typedef unique_ptr<LocalFunctionData> (*copy_to_initialize_local_t)(ExecutionContext &context, FunctionData &bind_data);
typedef unique_ptr<GlobalFunctionData> (*copy_to_initialize_global_t)(ClientContext &context, FunctionData &bind_data,
                                                                      const string &file_path);
typedef void (*copy_to_sink_t)(ExecutionContext &context, FunctionData &bind_data, GlobalFunctionData &gstate,
                               LocalFunctionData &lstate, DataChunk &input);
typedef void (*copy_to_combine_t)(ExecutionContext &context, FunctionData &bind_data, GlobalFunctionData &gstate,
                                  LocalFunctionData &lstate);
typedef void (*copy_to_finalize_t)(ClientContext &context, FunctionData &bind_data, GlobalFunctionData &gstate);

typedef void (*copy_to_serialize_t)(FieldWriter &writer, const FunctionData &bind_data, const CopyFunction &function);

typedef unique_ptr<FunctionData> (*copy_to_deserialize_t)(ClientContext &context, FieldReader &reader,
                                                          CopyFunction &function);

typedef unique_ptr<FunctionData> (*copy_from_bind_t)(ClientContext &context, CopyInfo &info,
                                                     vector<string> &expected_names,
                                                     vector<LogicalType> &expected_types);
typedef CopyFunctionExecutionMode (*copy_to_execution_mode_t)(bool preserve_insertion_order, bool supports_batch_index);

typedef unique_ptr<PreparedBatchData> (*copy_prepare_batch_t)(ClientContext &context, FunctionData &bind_data,
                                                              GlobalFunctionData &gstate,
                                                              unique_ptr<ColumnDataCollection> collection);
typedef void (*copy_flush_batch_t)(ClientContext &context, FunctionData &bind_data, GlobalFunctionData &gstate,
                                   PreparedBatchData &batch);
typedef idx_t (*copy_desired_batch_size_t)(ClientContext &context, FunctionData &bind_data);

class CopyFunction : public Function {
public:
	explicit CopyFunction(string name)
	    : Function(name), plan(nullptr), copy_to_bind(nullptr), copy_to_initialize_local(nullptr),
	      copy_to_initialize_global(nullptr), copy_to_sink(nullptr), copy_to_combine(nullptr),
	      copy_to_finalize(nullptr), execution_mode(nullptr), prepare_batch(nullptr), flush_batch(nullptr),
	      desired_batch_size(nullptr), serialize(nullptr), deserialize(nullptr), copy_from_bind(nullptr) {
	}

	//! Plan rewrite copy function
	copy_to_plan_t plan;

	copy_to_bind_t copy_to_bind;
	copy_to_initialize_local_t copy_to_initialize_local;
	copy_to_initialize_global_t copy_to_initialize_global;
	copy_to_sink_t copy_to_sink;
	copy_to_combine_t copy_to_combine;
	copy_to_finalize_t copy_to_finalize;
	copy_to_execution_mode_t execution_mode;

	copy_prepare_batch_t prepare_batch;
	copy_flush_batch_t flush_batch;
	copy_desired_batch_size_t desired_batch_size;

	copy_to_serialize_t serialize;
	copy_to_deserialize_t deserialize;

	copy_from_bind_t copy_from_bind;
	TableFunction copy_from_function;

	string extension;
};

} // namespace duckdb


namespace duckdb {

struct CreateCopyFunctionInfo : public CreateInfo {
	DUCKDB_API explicit CreateCopyFunctionInfo(CopyFunction function);

	//! Function name
	string name;
	//! The table function
	CopyFunction function;

protected:
	void SerializeInternal(Serializer &) const override;

public:
	unique_ptr<CreateInfo> Copy() const override;
};

} // namespace duckdb

#endif
